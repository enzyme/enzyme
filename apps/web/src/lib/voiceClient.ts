import { apiClient, throwIfError } from '@enzyme/api-client';
import type { SDPDescription, ICEServer } from '@enzyme/api-client';

export interface VoiceClientCallbacks {
  onConnectionStateChange?: (state: RTCPeerConnectionState) => void;
  onLocalSpeakingChange?: (speaking: boolean) => void;
}

export class VoiceClient {
  private pc: RTCPeerConnection | null = null;
  private localStream: MediaStream | null = null;
  private channelId: string;
  private callbacks: VoiceClientCallbacks;
  private audioElements = new Map<string, HTMLAudioElement>();
  private audioContext: AudioContext | null = null;
  private analyser: AnalyserNode | null = null;
  private speakingInterval: ReturnType<typeof setInterval> | null = null;
  private isSpeaking = false;

  constructor(channelId: string, callbacks: VoiceClientCallbacks = {}) {
    this.channelId = channelId;
    this.callbacks = callbacks;
  }

  async join(): Promise<void> {
    // Get microphone
    this.localStream = await navigator.mediaDevices.getUserMedia({ audio: true });

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

    // Set up local speaking detection
    this.startSpeakingDetection(this.localStream);

    // Handle remote tracks
    this.pc.ontrack = (event) => {
      const stream = event.streams[0];
      if (stream) {
        // Create audio element for playback
        const audio = document.createElement('audio');
        audio.srcObject = stream;
        audio.autoplay = true;
        this.audioElements.set(stream.id, audio);
      }
    };

    // Handle ICE candidates
    this.pc.onicecandidate = async (event) => {
      if (event.candidate) {
        await apiClient.POST('/channels/{id}/voice/ice-candidate', {
          params: { path: { id: this.channelId } },
          body: {
            candidate: event.candidate.candidate,
            sdp_mid: event.candidate.sdpMid || undefined,
            sdp_mline_index: event.candidate.sdpMLineIndex ?? undefined,
          },
        });
      }
    };

    this.pc.onconnectionstatechange = () => {
      if (this.pc) {
        this.callbacks.onConnectionStateChange?.(this.pc.connectionState);
      }
    };

    // Set remote description (server's offer)
    await this.pc.setRemoteDescription(
      new RTCSessionDescription({
        type: response.offer.type as RTCSdpType,
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
    // Notify server
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
          type: offer.type as RTCSdpType,
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
    // Mute all remote audio
    this.audioElements.forEach((audio) => {
      audio.muted = deafened;
    });
    apiClient
      .POST('/channels/{id}/voice/deafen', {
        params: { path: { id: this.channelId } },
        body: { deafened },
      })
      .catch(console.error);
  }

  private startSpeakingDetection(stream: MediaStream): void {
    this.audioContext = new AudioContext();
    this.analyser = this.audioContext.createAnalyser();
    this.analyser.fftSize = 512;
    this.analyser.smoothingTimeConstant = 0.4;

    const source = this.audioContext.createMediaStreamSource(stream);
    source.connect(this.analyser);

    const dataArray = new Uint8Array(this.analyser.frequencyBinCount);
    const SPEAKING_THRESHOLD = 15;
    const ACTIVATE_FRAMES = 3; // 150ms above threshold to activate
    const DEACTIVATE_FRAMES = 6; // 300ms below threshold to deactivate
    let consecutiveAbove = 0;
    let consecutiveBelow = 0;

    this.speakingInterval = setInterval(() => {
      if (!this.analyser) return;

      // Skip analysis when muted
      const track = stream.getAudioTracks()[0];
      if (track && !track.enabled) {
        if (this.isSpeaking) {
          this.isSpeaking = false;
          this.callbacks.onLocalSpeakingChange?.(false);
        }
        consecutiveAbove = 0;
        consecutiveBelow = 0;
        return;
      }

      this.analyser.getByteFrequencyData(dataArray);

      // Average the frequency data
      let sum = 0;
      for (let i = 0; i < dataArray.length; i++) {
        sum += dataArray[i];
      }
      const average = sum / dataArray.length;
      const aboveThreshold = average > SPEAKING_THRESHOLD;

      if (aboveThreshold) {
        consecutiveAbove++;
        consecutiveBelow = 0;
      } else {
        consecutiveBelow++;
        consecutiveAbove = 0;
      }

      if (!this.isSpeaking && consecutiveAbove >= ACTIVATE_FRAMES) {
        this.isSpeaking = true;
        this.callbacks.onLocalSpeakingChange?.(true);
      } else if (this.isSpeaking && consecutiveBelow >= DEACTIVATE_FRAMES) {
        this.isSpeaking = false;
        this.callbacks.onLocalSpeakingChange?.(false);
      }
    }, 50);
  }

  private stopSpeakingDetection(): void {
    if (this.speakingInterval) {
      clearInterval(this.speakingInterval);
      this.speakingInterval = null;
    }
    if (this.audioContext) {
      this.audioContext.close().catch(() => {});
      this.audioContext = null;
    }
    this.analyser = null;
    this.isSpeaking = false;
  }

  private cleanup(): void {
    // Stop speaking detection
    this.stopSpeakingDetection();

    // Stop local media tracks
    this.localStream?.getTracks().forEach((track) => track.stop());
    this.localStream = null;

    // Remove audio elements
    this.audioElements.forEach((audio) => {
      audio.srcObject = null;
    });
    this.audioElements.clear();

    // Close peer connection
    this.pc?.close();
    this.pc = null;
  }
}
