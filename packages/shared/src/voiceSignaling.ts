// Bridge between SSE voice events and the VoiceClient instance.
// useSSE dispatches events here; VoiceChannelView registers callbacks.

type OfferCallback = (data: { sdp: string; type: string }) => void;
type ICECandidateCallback = (data: {
  candidate: string;
  sdp_mid?: string;
  sdp_mline_index?: number;
}) => void;

let onOffer: OfferCallback | null = null;
let onICECandidate: ICECandidateCallback | null = null;

export function setVoiceSignalingCallbacks(callbacks: {
  onOffer?: OfferCallback;
  onICECandidate?: ICECandidateCallback;
}) {
  onOffer = callbacks.onOffer ?? null;
  onICECandidate = callbacks.onICECandidate ?? null;
}

export function clearVoiceSignalingCallbacks() {
  onOffer = null;
  onICECandidate = null;
}

export function dispatchVoiceOffer(data: { sdp: string; type: string }) {
  onOffer?.(data);
}

export function dispatchVoiceICECandidate(data: {
  candidate: string;
  sdp_mid?: string;
  sdp_mline_index?: number;
}) {
  onICECandidate?.(data);
}
