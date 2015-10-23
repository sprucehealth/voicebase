var objectAssign = require('object-assign');

module.exports = {
	ajax: function(params: ajaxParams, cb: ajaxCB, async?: bool) {
		jQuery.ajax(objectAssign(params, {
			async: (async == true || async == null),
			url: "/dermatologist-near-me/api" + params.url,
			success: function(data) {
				cb(true, data, noError, null);
			},
			error: function(jqXHR) {
				// Since success=false already is used to signal that data can be null
				// we can force flow to not throw errors on missing null checks on data.
				var x: any = null;
				cb(false, x, parseError(jqXHR), jqXHR);
			}
		}));
	},

	textDownloadLink: function(doctorID: string, number: string, cb: ajaxCB) {
		this.ajax({
			type: "POST",
			contentType: "application/json",
			url: "/textdownloadlink",
			data: JSON.stringify({doctor_id: doctorID, number: number}),
			dataType: "json"
		}, cb);
	},
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