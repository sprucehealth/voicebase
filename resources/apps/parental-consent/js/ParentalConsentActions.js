/* @flow */

var Reflux = require('reflux');
var ParentalConsentAPI = require("./api.js")

var Actions = Reflux.createActions({
  uploadGovernmentID: { sync: true, children: ['completed', 'failed'] }, // sync: true is a workaround for: https://github.com/spoike/refluxjs/issues/352
  uploadSelfie: { sync: true, children: ['completed', 'failed'] },
  saveDemographics: {},
  saveRelationship: {},
  saveEmailAndPassword: {},
  submitEmailRelationshipConsent: { asyncResult: true },
  signUp: { asyncResult: true },
  submitConsent: { asyncResult: true },
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

Actions.signUp.listen(function(request: ParentalConsentSignUpRequest) {
	var action = this
	ParentalConsentAPI.signUp(request, function(success, data, error) {
		if (!success) {
			action.failed(error)
		} else {
			action.completed(data)
		}
	});
});

Actions.submitConsent.listen(function(request: ParentalConsentConsentRequest) {
	var action = this
	ParentalConsentAPI.submitConsent(request, function(success, data, error) {
		if (!success) {
			action.failed(error)
		} else {
			action.completed(data)
		}
	});
});

module.exports = Actions;