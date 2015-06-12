type ajaxError = {message: string};

type jqXHR = {
	status: number;
	responseText: string;
};

declare function ajaxCB(success: bool, data: any, error: ?ajaxError, jqXHR?: ?jqXHR): void;

declare function ajaxSuccessCB(data: any): void;
declare function ajaxErrorCB(jqXHR: jqXHR): void;

type ajaxParams = {
	type: string;
	url: string;
	dataType: string;
} | {
	type: string;
	url: string;
	data: string;
	dataType: string;
	contentType: string;
} | {
	type: string;
	url: string;
	data: any;
	contentType: boolean | string;
	cache: boolean;
	processData: boolean;
};
