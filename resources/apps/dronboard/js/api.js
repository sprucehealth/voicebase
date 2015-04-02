/* @flow */

var objectAssign = require('object-assign');

module.exports = {
	// cb is function(success: bool, data: object, jqXHR: jqXHR)
	ajax: function(params: ajaxParams, cb: ajaxCB) {
		jQuery.ajax(objectAssign(params, {
			success: function(data) {
				cb(true, data, noError, null);
			},
			error: function(jqXHR) {
				cb(false, null, parseError(jqXHR), jqXHR);
			}
		}));
	},
	updateCellNumber: function(number: string, cb: ajaxCB) {
		this.ajax({
			type: "POST",
			contentType: "application/json",
			url: "/doctor-register/cell-verify",
			data: JSON.stringify({number: number}),
			dataType: "json"
		}, cb);
	},
	verifyCellNumber: function(number: string, code: string, cb: ajaxCB) {
		this.ajax({
			type: "POST",
			contentType: "application/json",
			url: "/doctor-register/cell-verify",
			data: JSON.stringify({number: number, code: code}),
			dataType: "json"
		}, cb);
	}
};

var noError: ajaxError = {message: ""};

function parseError(jqXHR: jqXHR): ajaxError {
	if (jqXHR.status == 0) {
		return {message: "Network request failed"};
	}
	var err;
	try {
		err = JSON.parse(jqXHR.responseText).error;
	} catch(e) {
		if (jqXHR.status == 403) {
			err = {message: "Access denied"};
		} else {
			console.error(jqXHR.responseText);
			err = {message: "Unknown error"};
		}
	}
	return err;
}
