
/*
* The Fetch polyfill by Github offers promise-based asynchronous networking calls
*
* We need to declare it here because it's actually included globally as a Webpack plugin, rather than being imported individually in files.
*/
declare function fetch(path: any, data: any): any;

declare var Spruce: {
	BaseStaticURL: string;
};