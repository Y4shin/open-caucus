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
	stream.addEventListener('attendees.updated', handleEvent);
	stream.addEventListener('speakers.updated', handleEvent);
	stream.addEventListener('agenda.updated', handleEvent);
	stream.addEventListener('votes.updated', handleEvent);

	return () => {
		stream.close();
	};
}
