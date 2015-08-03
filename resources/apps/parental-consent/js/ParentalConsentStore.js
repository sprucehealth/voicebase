/* @flow */

var Reflux = require('reflux')
var ParentalConsentAPI = require("./api.js")
var Utils = require("../../libs/utils.js");

var ParentalConsentActions = require('./ParentalConsentActions.js')

var hydration = {
	ChildDetails: {},
	ParentalConsent: {
		consented: false,
	},
	IdentityVerificationImages: {
		types: {
			governmentid: "",
			selfie: "",
		}
	},
	IsParentSignedIn: false,
};
if (typeof ParentalConsentHydration != "undefined") {
	hydration = ParentalConsentHydration;
}

var PhotoIdentificationAlreadySubmittedAtPageLoad = false
if (hydration.IdentityVerificationImages) {
	var IdentityVerificationImages: ParentalConsentGetImagesResponse = hydration.IdentityVerificationImages
	PhotoIdentificationAlreadySubmittedAtPageLoad = !Utils.isEmpty(IdentityVerificationImages.types.governmentid) && !Utils.isEmpty(IdentityVerificationImages.types.selfie)
}

var possessivePronoun: string = "their"
if (hydration.ChildDetails.gender === "male") {
	possessivePronoun = "his"
} else if (hydration.ChildDetails.gender === "female") {
	possessivePronoun = "her"
}

var personalPronoun: string = hydration.ChildDetails.firstName // using first name is intentional: "he/she is now able to..." -> "Jimmy is now able to..."
if (hydration.ChildDetails.gender === "male") {
	personalPronoun = "he"
} else if (hydration.ChildDetails.gender === "female") {
	personalPronoun = "she"
}
if (!personalPronoun) {
	personalPronoun = "they";
}

var externalState: ParentalConsentStoreType = {
	Token: "",
	childDetails: {
		firstName: hydration.ChildDetails.firstName,
		possessivePronoun: possessivePronoun,
		personalPronoun: personalPronoun,
		childPatientID: hydration.ChildDetails.patientID,
	},
	PhotoIdentificationAlreadySubmittedAtPageLoad: PhotoIdentificationAlreadySubmittedAtPageLoad,
	ConsentWasAlreadySubmittedAtPageLoad: hydration.ParentalConsent.consented,
	parentAccount: {
		WasSignedInAtPageLoad: hydration.IsParentSignedIn,
		isSignedIn: hydration.IsParentSignedIn,
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
		consents: {
			consentedToConsentToUseOfTelehealth: hydration.ParentalConsent.consented,
			consentedToTermsAndPrivacy: hydration.ParentalConsent.consented,
		},
	},
	identityVerification: {
		serverGovernmentIDThumbnailURL: hydration.IdentityVerificationImages.types.governmentid,
		serverSelfieThumbnailURL: hydration.IdentityVerificationImages.types.selfie,
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

		// NOTE: this is an inaccurate way to do this (better: use a library), but since the server is also checking age, we'll get by
		// From: http://stackoverflow.com/questions/4060004/calculate-age-in-javascript/7091965#7091965
		function getAge(dateString) {
			var today = new Date();
			var birthDate = new Date(dateString);
			var age = today.getFullYear() - birthDate.getFullYear();
			var m = today.getMonth() - birthDate.getMonth();
			if (m < 0 || (m === 0 && today.getDate() < birthDate.getDate())) {
				age--;
			}
			return age;
		}

		if (this.validateMMDDYY(demographics.dob)) {
			var YYYYMMDD = this.YYYYMMDDFromMMDDYY(demographics.dob)
			if (getAge(YYYYMMDD) >= 18) {
				externalState.userInput.demographics = demographics
				this.trigger(externalState)
				ParentalConsentActions.saveDemographics.completed()
			} else {
				ParentalConsentActions.saveDemographics.failed({message: "You must be 18 years or older."})
			}
		} else {
			ParentalConsentActions.saveDemographics.failed({message: "Please enter a valid date of birth."})
		}
	},
	validateMMDDYY: function(MMDDYY: string): bool {
		var regex = new RegExp(/^((0?[13578]|10|12)(-|\/)(([1-9])|(0[1-9])|([12])([0-9]?)|(3[01]?))(-|\/)((19)([2-9])(\d{1})|(20)([01])(\d{1})|([8901])(\d{1}))|(0?[2469]|11)(-|\/)(([1-9])|(0[1-9])|([12])([0-9]?)|(3[0]?))(-|\/)((19)([2-9])(\d{1})|(20)([01])(\d{1})|([8901])(\d{1})))$/)
		return regex.test(MMDDYY)
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
	YYYYMMDDFromMMDDYY: function(MMDDYY: string): string {
		var dateIsValid = this.validateMMDDYY(MMDDYY)
		var YYYYMMDD: string = ""
		if (dateIsValid) {
			var components = MMDDYY.split("-");
			// Not Y2.1k safe
			var YYYY
			var d = new Date()
			var currentYYInt: number = d.getFullYear() % 100
			if (parseInt(components[2]) > currentYYInt) {
				YYYY = "19" + components[2]
			} else {
				YYYY = "20" + components[2]
			}
			YYYYMMDD = YYYY + "-" + components[0] + "-" + components[1]
		}
		return YYYYMMDD
	},
	onSaveEmailAndPassword: function(emailPassword: ParentalConsentEmailPassword) {
		externalState.userInput.emailPassword = emailPassword
	},
	onSubmitEmailRelationshipConsent: function(unused) {

		var userInput: ParentalConsentAllUserInput = externalState.userInput
		var relationship: any = userInput.relationship

		var consentRequest: ParentalConsentConsentRequest = {
			child_patient_id: externalState.childDetails.childPatientID,
			relationship: relationship,
		}

		var YYYYMMDD: string = this.YYYYMMDDFromMMDDYY(userInput.demographics.dob)

		var signUpRequest: ParentalConsentSignUpRequest = {
			email: userInput.emailPassword.email,
			password: userInput.emailPassword.password,
			state: userInput.demographics.state,
			first_name: userInput.demographics.first_name,
			last_name: userInput.demographics.last_name,
			dob: YYYYMMDD,
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
			if (Utils.isEmpty(YYYYMMDD)) {
				ParentalConsentActions.submitEmailRelationshipConsent.failed({message: "Please provide a valid date of birth."})
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
	}
});

module.exports = ParentalConsentStore;