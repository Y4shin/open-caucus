export function connectEventStream(url: string, onInvalidate: () => void): () => void {
	const stream = new EventSource(url, { withCredentials: true });

	const handleEvent = (event: MessageEvent) => {
		if (event.type !== 'connected') {
			onInvalidate();
		}
	};

	stream.onmessage = handleEvent;
	stream.addEventListener('connected', handleEvent);
	stream.addEventListener('moderate-updated', handleEvent);
	stream.addEventListener('live-updated', handleEvent);

	return () => {
		stream.close();
	};
}
