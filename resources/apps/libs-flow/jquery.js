declare module jQuery {
	declare function ajax(params: any, cb: any): void;
	declare function modal(action: string): void;
}

declare var jQuery: jQuery;
declare var $: jQuery;
