import {
  RTCPeerConnection,
  RTCSessionDescription,
  RTCIceCandidate,
  mediaDevices,
  MediaStream,
} from 'react-native-webrtc';
import { apiClient, throwIfError } from '@enzyme/api-client';
import type { SDPDescription, ICEServer } from '@enzyme/api-client';

export interface VoiceClientCallbacks {
  onConnectionStateChange?: (state: string) => void;
}

export class VoiceClient {
  private pc: RTCPeerConnection | null = null;
  private localStream: MediaStream | null = null;
  private channelId: string;
  private callbacks: VoiceClientCallbacks;

  constructor(channelId: string, callbacks: VoiceClientCallbacks = {}) {
    this.channelId = channelId;
    this.callbacks = callbacks;
  }

  async join(): Promise<void> {
    // Get microphone
    this.localStream = (await mediaDevices.getUserMedia({ audio: true })) as MediaStream;

    // Call join endpoint
    const response = await throwIfError(
      apiClient.POST('/channels/{id}/voice/join', {
        params: { path: { id: this.channelId } },
      }),
    );

    // Create peer connection with ICE servers from the server
    const iceServers = response.ice_servers.map((s: ICEServer) => ({
      urls: s.urls,
      username: s.username,
      credential: s.credential,
    }));

    this.pc = new RTCPeerConnection({ iceServers });

    // Add local audio track
    this.localStream.getAudioTracks().forEach((track) => {
      this.pc!.addTrack(track, this.localStream!);
    });

    // Handle ICE candidates — react-native-webrtc extends EventTarget from
    // event-target-shim; cast to any for addEventListener compatibility.
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const pc = this.pc as any;
    pc.addEventListener('icecandidate', (event: { candidate: RTCIceCandidate | null }) => {
      if (event.candidate) {
        apiClient.POST('/channels/{id}/voice/ice-candidate', {
          params: { path: { id: this.channelId } },
          body: {
            candidate: event.candidate.candidate,
            sdp_mid: event.candidate.sdpMid || undefined,
            sdp_mline_index: event.candidate.sdpMLineIndex ?? undefined,
          },
        });
      }
    });

    pc.addEventListener('connectionstatechange', () => {
      if (this.pc) {
        this.callbacks.onConnectionStateChange?.(this.pc.connectionState);
      }
    });

    // Set remote description (server's offer)
    await this.pc.setRemoteDescription(
      new RTCSessionDescription({
        type: response.offer.type as 'offer',
        sdp: response.offer.sdp,
      }),
    );

    // Create and send answer
    const answer = await this.pc.createAnswer();
    await this.pc.setLocalDescription(answer);

    await apiClient.POST('/channels/{id}/voice/answer', {
      params: { path: { id: this.channelId } },
      body: {
        answer: {
          sdp: answer.sdp!,
          type: answer.type as SDPDescription['type'],
        },
      },
    });
  }

  async leave(): Promise<void> {
    await apiClient
      .POST('/channels/{id}/voice/leave', {
        params: { path: { id: this.channelId } },
      })
      .catch(() => {}); // best-effort

    this.cleanup();
  }

  handleRemoteOffer(offer: { sdp: string; type: string }): void {
    if (!this.pc) return;

    this.pc
      .setRemoteDescription(
        new RTCSessionDescription({
          type: offer.type as 'offer',
          sdp: offer.sdp,
        }),
      )
      .then(() => {
        if (!this.pc) return;
        return this.pc.createAnswer();
      })
      .then((answer) => {
        if (!this.pc || !answer) return;
        this.pc.setLocalDescription(answer);
        return apiClient.POST('/channels/{id}/voice/answer', {
          params: { path: { id: this.channelId } },
          body: {
            answer: {
              sdp: answer.sdp!,
              type: answer.type as SDPDescription['type'],
            },
          },
        });
      })
      .catch(console.error);
  }

  handleRemoteICECandidate(candidate: {
    candidate: string;
    sdp_mid?: string;
    sdp_mline_index?: number;
  }): void {
    if (!this.pc) return;

    this.pc
      .addIceCandidate(
        new RTCIceCandidate({
          candidate: candidate.candidate,
          sdpMid: candidate.sdp_mid,
          sdpMLineIndex: candidate.sdp_mline_index,
        }),
      )
      .catch(console.error);
  }

  setMuted(muted: boolean): void {
    if (this.localStream) {
      this.localStream.getAudioTracks().forEach((track) => {
        track.enabled = !muted;
      });
    }
    apiClient
      .POST('/channels/{id}/voice/mute', {
        params: { path: { id: this.channelId } },
        body: { muted },
      })
      .catch(console.error);
  }

  setDeafened(deafened: boolean): void {
    // Mute/unmute all remote audio tracks
    if (this.pc) {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      (this.pc as any).getReceivers().forEach((receiver: any) => {
        if (receiver.track) {
          receiver.track.enabled = !deafened;
        }
      });
    }
    apiClient
      .POST('/channels/{id}/voice/deafen', {
        params: { path: { id: this.channelId } },
        body: { deafened },
      })
      .catch(console.error);
  }

  isConnected(): boolean {
    if (!this.pc) return false;
    const state = this.pc.connectionState;
    return state === 'connected' || state === 'connecting' || state === 'new';
  }

  private cleanup(): void {
    this.localStream?.getTracks().forEach((track) => track.stop());
    this.localStream = null;
    this.pc?.close();
    this.pc = null;
  }
}
