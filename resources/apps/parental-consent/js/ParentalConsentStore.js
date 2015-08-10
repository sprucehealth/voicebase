/* @flow */

var Reflux = require('reflux')
var Utils = require("../../libs/utils.js");
var ParentalConsentAPI = require("./api.js")
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

		var phone = (demographics.mobile_phone ? demographics.mobile_phone.replace(/\D/g,'') : "");
		var phoneIsValid = phone
			&& !Utils.isEmpty(phone)
			&& (phone.length === 10 || (phone.length === 11 && phone.substring(0, 1) === "1"))
		if (!phoneIsValid) {
			ParentalConsentActions.saveDemographics.failed({message: "Please enter a 10-digit phone number (ex: 415-555-1212)."})
			return;
		}

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

		if (this.validateUserInputDOB(demographics.dob)) {
			var YYYYMMDD = this.YYYYMMDDFromUserInputDOB(demographics.dob)
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
	validateUserInputDOB: function(userInputDOB: string): bool {
		return (!Utils.isEmpty(userInputDOB) && (userInputDOB.length == 8 || userInputDOB.length == 10))
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
	YYYYfromYY: function(YY: string): string {
		// Not Y2.1k safe
		var YYYY
		var d = new Date()
		var currentYYInt: number = d.getFullYear() % 100
		if (parseInt(YY) > currentYYInt) {
			YYYY = "19" + YY
		} else {
			YYYY = "20" + YY
		}
		return YYYY
	},
	YYYYMMDDFromUserInputDOB: function(userInputDOB: string): string {
		var dateIsValid = this.validateUserInputDOB(userInputDOB)
		var YYYYMMDD: string = ""
		if (dateIsValid) {
			var components = userInputDOB.split("-");

			var YYYY = ""
			var MM = ""
			var DD = ""
			if (components.length == 3 && components[0].length === 2) {
				if (components && components[2] && components[2].length == 2) {
					YYYY = this.YYYYfromYY(components[2])
				} else if (components && components[2] && components[2].length == 4) {
					YYYY = components[2]
				}
				MM = components[0] 
				DD = components[1]
			} else if (components.length > 0 && components[0].length === 4) {
				YYYY = components[0]
				MM = components[1] 
				DD = components[2]
			}
				
			YYYYMMDD = YYYY + "-" + MM + "-" + DD
			var date = new Date(YYYYMMDD)
			if (date === null) {
				YYYYMMDD = ""
			}
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
			relationship: relationship.trim(),
		}

		var YYYYMMDD: string = this.YYYYMMDDFromUserInputDOB(userInput.demographics.dob)

		var signUpRequest: ParentalConsentSignUpRequest = {
			email: userInput.emailPassword.email.trim(),
			password: userInput.emailPassword.password.trim(),
			state: userInput.demographics.state.trim(),
			first_name: userInput.demographics.first_name.trim(),
			last_name: userInput.demographics.last_name.trim(),
			dob: YYYYMMDD,
			gender: userInput.demographics.gender.trim(),
			mobile_phone: userInput.demographics.mobile_phone.trim(),
		}

		var t = this

		var submitConsent = function () {
			externalState.numBlockingOperations = externalState.numBlockingOperations + 1
			t.trigger(externalState)

			ParentalConsentAPI.submitConsent(consentRequest, function(success, data, error) {
				if (!success) {
					externalState.numBlockingOperations = externalState.numBlockingOperations - 1
					t.trigger(externalState)
					ParentalConsentActions.submitEmailRelationshipConsent.failed(error)
				} else {
					externalState.userInput.consents.consentedToConsentToUseOfTelehealth = true
					externalState.userInput.consents.consentedToTermsAndPrivacy = true
					externalState.numBlockingOperations = externalState.numBlockingOperations - 1
					t.trigger(externalState)
					ParentalConsentActions.submitEmailRelationshipConsent.completed()
				}
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
				ParentalConsentAPI.signUp(signUpRequest, function(success, data, error) {
					if (!success) {
						externalState.numBlockingOperations = externalState.numBlockingOperations - 1
						t.trigger(externalState)
						ParentalConsentActions.submitEmailRelationshipConsent.failed(error)
					} else {
						externalState.parentAccount.isSignedIn = true;
						externalState.numBlockingOperations = externalState.numBlockingOperations - 1
					    t.trigger(externalState)
					    submitConsent()
					}
				});
			}
		}
	}
});

module.exports = ParentalConsentStore;