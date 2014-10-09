module.exports = {
	// cb is function(success: bool, data: object, jqXHR: jqXHR)
	ajax: function(params, cb) {
		params.success = function(data) {
			cb(true, data, null);
		}
		params.error = function(jqXHR) {
			cb(false, null, jqXHR);
		}
		jQuery.ajax(params);
	},
	updateCellNumber: function(number, cb) {
		this.ajax({
			type: "POST",
			contentType: "application/json",
			url: "/doctor-register/cell-verify",
			data: JSON.stringify({number: number}),
			dataType: "json"
		}, cb);
	},
	verifyCellNumber: function(number, code, cb) {
		this.ajax({
			type: "POST",
			contentType: "application/json",
			url: "/doctor-register/cell-verify",
			data: JSON.stringify({number: number, code: code}),
			dataType: "json"
		}, cb);
	},
	parseError: function(jqXHR) {
		var err;
		try {
			err = JSON.parse(jqXHR.responseText)
		} catch(e) {
			err = {error: jqXHR.responseText};
		}
		return err;
	}
};
