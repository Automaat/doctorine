import {
	parseCookie,
	stringifySetCookie,
	type ParseOptions,
	type SerializeOptions,
	type StringifyOptions
} from 'cookie';

export function parse(str: string, options?: ParseOptions) {
	return parseCookie(str, options);
}

export function serialize(name: string, value: string, options: SerializeOptions = {}) {
	const { encode, ...attributes } = options;
	const stringifyOptions: StringifyOptions | undefined = encode ? { encode } : undefined;

	return stringifySetCookie({ name, value, ...attributes }, stringifyOptions);
}
