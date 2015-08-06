/* @flow */

var Reflux = require('reflux');
var ParentalConsentAPI = require("./api.js")

var Actions = Reflux.createActions({
	uploadGovernmentID: { sync: true, children: ['completed', 'failed'] }, // sync: true is a workaround for: https://github.com/spoike/refluxjs/issues/352
	uploadSelfie: { sync: true, children: ['completed', 'failed'] },
	saveDemographics: { sync: true, children: ['completed', 'failed']},
	saveRelationship: {},
	saveEmailAndPassword: {},
	submitEmailRelationshipConsent: { asyncResult: true },
});

Actions.uploadGovernmentID.listen(function(formData: FormData) {
	var action = this
	ParentalConsentAPI.uploadPhoto(formData, function(success, data, error) {
		if (!success) {
			action.failed(error)
		} else {
			action.completed(data)
		}
	});
});

Actions.uploadSelfie.listen(function(formData: FormData) {
	var action = this
	ParentalConsentAPI.uploadPhoto(formData, function(success, data, error) {
		if (!success) {
			action.failed(error)
		} else {
			action.completed(data)
		}
	});
});

module.exports = Actions;