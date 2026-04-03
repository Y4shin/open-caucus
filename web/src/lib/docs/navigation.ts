export const DOCS_PATH_PARAM = 'docs';
export const DOCS_HEADING_PARAM = 'docs_heading';
export const DOCS_QUERY_PARAM = 'docs_q';

type DocsLinkOptions = {
	heading?: string;
	query?: string;
};

export function normalizeDocsPath(path: string): string {
	const trimmed = path.trim();
	if (!trimmed) return 'index';
	return trimmed.replace(/^\/+/, '').replace(/^docs\/+/, '').replace(/^\/+/, '');
}

export function splitDocsRef(ref: string): { path: string; heading: string } {
	const [rawPath, rawHeading = ''] = ref.split('#', 2);
	return {
		path: normalizeDocsPath(rawPath),
		heading: rawHeading.trim()
	};
}

export function buildStandaloneDocsHref(refOrPath: string, options: DocsLinkOptions = {}): string {
	const { path, heading: refHeading } = splitDocsRef(refOrPath);
	const url = new URL(`/docs/${path}`, 'https://docs.invalid');
	const heading = options.heading?.trim() || refHeading;
	const query = options.query?.trim() ?? '';
	if (heading) {
		url.searchParams.set('heading', heading);
	}
	if (query) {
		url.searchParams.set('q', query);
	}
	return `${url.pathname}${url.search}`;
}

export function buildDocsOverlayHref(refOrPath: string, currentUrl: URL, options: DocsLinkOptions = {}): string {
	const { path, heading: refHeading } = splitDocsRef(refOrPath);
	const url = new URL(currentUrl.toString());
	url.searchParams.set(DOCS_PATH_PARAM, path);
	const heading = options.heading?.trim() || refHeading;
	const query = options.query?.trim() ?? '';
	if (heading) {
		url.searchParams.set(DOCS_HEADING_PARAM, heading);
	} else {
		url.searchParams.delete(DOCS_HEADING_PARAM);
	}
	if (query) {
		url.searchParams.set(DOCS_QUERY_PARAM, query);
	} else {
		url.searchParams.delete(DOCS_QUERY_PARAM);
	}
	return `${url.pathname}${url.search}${url.hash}`;
}

export function clearDocsOverlayHref(currentUrl: URL): string {
	const url = new URL(currentUrl.toString());
	url.searchParams.delete(DOCS_PATH_PARAM);
	url.searchParams.delete(DOCS_HEADING_PARAM);
	url.searchParams.delete(DOCS_QUERY_PARAM);
	return `${url.pathname}${url.search}${url.hash}`;
}
