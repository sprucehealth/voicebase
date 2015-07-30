/* @flow */

var jQuery = require('jquery');
var objectAssign = require('object-assign');

module.exports = {
	StatusNotFound: 404,

	ajax: function(params: ajaxParams, cb: ajaxCB, async?: bool) {
		jQuery.ajax(objectAssign(params, {
			async: (async == true || async == null),
			url: "/api" + params.url,
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

	uploadPhoto: function(formData: any, cb: ajaxCB) {
		this.ajax({
			type: "POST",
			cache: false,
			contentType: false,
			processData: false,
			url: "/parental-consent/image",
			data: formData
		}, cb);
	},

	getParentalConsentImages: function(formData: any, cb: ajaxCB) {
		// TODO: remove before shipping
		// setTimeout(function() {
		// 	cb(false, {}, {message: "sample error"})
		// }, 1000);

		// TODO: remove before shipping
		var mockResponse: ParentalConsentGetImagesResponse = {
			types: {
				governmentid: "http://www.nicenicejpg.com/401/601",
				selfie: "http://www.nicenicejpg.com/600/400",
			}
		}
		setTimeout(function() {
			cb(true, mockResponse, null)
		}, 1000);

		//
		// TODO: uncomment this!!!
		//
		// this.ajax({
		// 	type: "GET",
		// 	cache: false,
		// 	contentType: false,
		// 	processData: false,
		// 	url: "/parental-consent/image/",
		// 	data: formData
		// }, cb);
	},

	signUp: function(request: ParentalConsentSignUpRequest, cb: ajaxCB) {
		this.ajax({
			type: "POST",
			contentType: "application/json",
			url: "/auth/sign-up",
			data: JSON.stringify(request),
			dataType: "json"
		}, cb);
	},

	signIn: function(request: ParentalConsentSignInRequest, cb: ajaxCB) {
		// TODO: remove before shipping
		// setTimeout(function() {
		// 	cb(false, {}, {message: "sample error"})
		// }, 1000);

		// TODO: remove before shipping
		setTimeout(function() {
			cb(true, {}, null)
		}, 1000);

		//
		// TODO: uncomment this!!!
		//
		// this.ajax({
		// 	type: "POST",
		// 	contentType: "application/json",
		// 	url: "/auth/sign-in",
		// 	data: JSON.stringify(request),
		// 	dataType: "json"
		// }, cb);
	},

	submitConsent: function(request: ParentalConsentConsentRequest, cb: ajaxCB) {
		this.ajax({
			type: "POST",
			contentType: "application/json",
			url: "/parental-consent",
			data: JSON.stringify(request),
			dataType: "json"
		}, cb);
	},

	getConsentStatus: function(request: any, cb: ajaxCB) {
		this.ajax({
			type: "GET",
			contentType: "application/json",
			url: "/parental-consent",
			data: JSON.stringify(request),
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
