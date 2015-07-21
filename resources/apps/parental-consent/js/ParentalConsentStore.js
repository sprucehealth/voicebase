/* @flow */

var Reflux = require('reflux')
var ParentalConsentAPI = require("./api.js")
var Utils = require("../../libs/utils.js");

var ParentalConsentActions = require('./ParentalConsentActions.js')

// TODO:
// Before shipping!!!
// Insert actual token

var PhotoIdentificationAlreadySubmittedAtPageLoad = false
if (ParentalConsentHydration.IdentityVerificationImages) {
	var IdentityVerificationImages: ParentalConsentGetImagesResponse = ParentalConsentHydration.IdentityVerificationImages
	PhotoIdentificationAlreadySubmittedAtPageLoad = !Utils.isEmpty(IdentityVerificationImages.types.governmentid) && !Utils.isEmpty(IdentityVerificationImages.types.selfie)
}

var possessivePronoun: string = "their"
if (ParentalConsentHydration.ChildDetails.gender === "male") {
	possessivePronoun = "his"
} else if (ParentalConsentHydration.ChildDetails.gender === "female") {
	possessivePronoun = "her"
}

var personalPronoun: string = ParentalConsentHydration.ChildDetails.firstName // using first name is intentional: "he/she is now able to..." -> "Jimmy is now able to..."
if (ParentalConsentHydration.ChildDetails.gender === "male") {
	personalPronoun = "he"
} else if (ParentalConsentHydration.ChildDetails.gender === "female") {
	personalPronoun = "she"
}

var externalState: ParentalConsentStoreType = {
	Token: "",
	childDetails: {
		firstName: ParentalConsentHydration.ChildDetails.firstName,
		possessivePronoun: possessivePronoun,
		personalPronoun: personalPronoun,
		childPatientID: ParentalConsentHydration.ChildDetails.patientID,
	},
	PhotoIdentificationAlreadySubmittedAtPageLoad: PhotoIdentificationAlreadySubmittedAtPageLoad,
	ConsentWasAlreadySubmittedAtPageLoad: ParentalConsentHydration.ParentalConsent.consented,
	parentAccount: {
		WasSignedInAtPageLoad: ParentalConsentHydration.IsParentSignedIn,
		isSignedIn: ParentalConsentHydration.IsParentSignedIn,
	},
	userInput: {
		emailPassword: {
			email: "",
			password: "",
		},
		demographics: {
			first_name: "",
			last_name: "",
			dob: "",
			gender: "",
			state: "",
			mobile_phone: "",
		},
		relationship: "",
	},
	consent: ParentalConsentHydration.ParentalConsent,
	identityVerification: {
		serverGovernmentIDThumbnailURL: ParentalConsentHydration.IdentityVerificationImages.types.governmentid,
		serverSelfieThumbnailURL: ParentalConsentHydration.IdentityVerificationImages.types.selfie,
	},
	numBlockingOperations: 0,
};

var ParentalConsentStore = Reflux.createStore({
	listenables: [ParentalConsentActions],
	init: function() { },
	getInitialState: function(): ParentalConsentStoreType {
		return externalState
	},

	// JS: I have a feeling it is a bit of an anti-pattern to expose state like this, but I can't currently figure out a way around it
	getCurrentState: function(): ParentalConsentStoreType {
		return externalState
	},

	//
	// Photo Verification
	//
	onUploadGovernmentID: function(unused) {},
	onUploadGovernmentIDCompleted: function(response: ParentalConsentUploadImageResponse) {
		// TODO: ASSERT that result.url is non-empty
		externalState.identityVerification.serverGovernmentIDThumbnailURL = response.url
		this.trigger(externalState)
	},
	onUploadGovernmentIDFailed: function(error) {
		// TODO: don't just clear out the image if it fails-- instead retry
		externalState.identityVerification.serverGovernmentIDThumbnailURL = ""
		this.trigger(externalState)
	},
	onUploadSelfie: function(unused) {},
	onUploadSelfieCompleted: function(response: ParentalConsentUploadImageResponse) {
		// TODO: ASSERT that result.url is non-empty
		externalState.identityVerification.serverSelfieThumbnailURL = response.url
		this.trigger(externalState)
	},
	onUploadSelfieFailed: function(error) {
		// TODO: don't just clear out the image if it fails-- instead retry
		externalState.identityVerification.serverSelfieThumbnailURL = ""
		this.trigger(externalState)
	},

	//
	// Demographics screen
	//
	onSaveDemographics: function(demographics: ParentalConsentDemographics) {
		externalState.userInput.demographics = demographics
		this.trigger(externalState)
	},

	//
	// Demographics OR Relationship screen
	//
	onSaveRelationship: function(relationship: string) {
		externalState.userInput.relationship = relationship
	},

	//
	// Email/Relationship/Consent screen
	//
	onSaveEmailAndPassword: function(emailPassword: ParentalConsentEmailPassword) {
		externalState.userInput.emailPassword = emailPassword
	},
	onSubmitEmailRelationshipConsent: function(unused) {

		var userInput: ParentalConsentAllUserInput = externalState.userInput
		var consent: any = externalState.consent
		var relationship: any = Utils.isEmpty(consent.relationship) ? userInput.relationship : consent.relationship

		var childDetails: any = externalState.childDetails.childPatientID
		var consentRequest: ParentalConsentConsentRequest = {
			child_patient_id: childDetails.patientID,
			relationship: relationship,
			token: externalState.Token,
		}

		var signUpRequest: ParentalConsentSignUpRequest = {
			email: userInput.emailPassword.email,
			password: userInput.emailPassword.password,
			state: userInput.demographics.state,
			first_name: userInput.demographics.first_name,
			last_name: userInput.demographics.last_name,
			dob: userInput.demographics.dob,
			gender: userInput.demographics.gender,
			mobile_phone: userInput.demographics.mobile_phone,
		}

		var t = this

		var submitConsent = function () {
			externalState.numBlockingOperations = externalState.numBlockingOperations + 1
			t.trigger(externalState)
			ParentalConsentActions.submitConsent.triggerPromise(consentRequest).then(function(response: any) {
				externalState.numBlockingOperations = externalState.numBlockingOperations - 1
				t.trigger(externalState)
				ParentalConsentActions.submitEmailRelationshipConsent.completed()
			}).catch(function(err: ajaxError) {
				externalState.numBlockingOperations = externalState.numBlockingOperations - 1
				t.trigger(externalState)
				ParentalConsentActions.submitEmailRelationshipConsent.failed(err)
			});
		}

		if (externalState.parentAccount.isSignedIn) {
		    submitConsent()
		} else {
			externalState.numBlockingOperations = externalState.numBlockingOperations + 1
			t.trigger(externalState)
			ParentalConsentActions.signUp.triggerPromise(signUpRequest).then(function(response: any) {
				externalState.parentAccount.isSignedIn = true;
				externalState.numBlockingOperations = externalState.numBlockingOperations - 1
			    submitConsent()
			    t.trigger(externalState)
			}).catch(function(err: ajaxError) {
				externalState.numBlockingOperations = externalState.numBlockingOperations - 1
				t.trigger(externalState)
				ParentalConsentActions.submitEmailRelationshipConsent.failed(err)
			});
		}
	}
});

module.exports = ParentalConsentStore;