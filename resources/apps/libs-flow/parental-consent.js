type ParentalConsentDemographics = {
	first_name: string;
	last_name: string;
	dob: string;
	gender: string;
	state: string;
	mobile_phone: string;
};


type ParentalConsentEmailPassword = {
	email: string;
	password: string;
};

type ParentalConsentSignUpRequest = {
	email: string;
	password: string;
	state: string;
	first_name: string;
	last_name: string;
	dob: string;
	gender: string;
	mobile_phone: string;
};

type ParentalConsentSignInRequest = {
	email: string;
	password: string;
};

type ParentalConsentConsentRequest = {
	child_patient_id: string;
	relationship: string;
};

type ParentalConsentConsentResponse = {
	"consented": bool;
	"relationship": ?string;
};

type ParentalConsentGetImagesResponse = {
	"types": {
		"governmentid": string;
		"selfie": string;
	};
};

type ParentalConsentUploadImageResponse = {
	"url": string;
};

type ParentalConsentAllUserInput = {
	emailPassword: ParentalConsentEmailPassword;
	demographics: ParentalConsentDemographics;
	relationship: string;
	consents: {
		consentedToTermsAndPrivacy: bool;
		consentedToConsentToUseOfTelehealth: bool;
	};
};

type ParentalConsentStoreType = {
	childDetails: {
		firstName: string;
		possessivePronoun: string;
		personalPronoun: string;
		childPatientID: string;
	};
	PhotoIdentificationAlreadySubmittedAtPageLoad: bool;
	ConsentWasAlreadySubmittedAtPageLoad: bool;
	parentAccount: {
		WasSignedInAtPageLoad: bool;
		isSignedIn: bool;
	};
	userInput: ParentalConsentAllUserInput;
	identityVerification: {
		serverGovernmentIDThumbnailURL: string;
		serverSelfieThumbnailURL: string;
	};
	numBlockingOperations: number;
};

declare var ParentalConsentHydration: {
	ChildDetails: {
		firstName: string;
		gender: string;
		patientID: string
	};
	ParentalConsent: ParentalConsentConsentResponse;
	IsParentSignedIn: bool;
	IdentityVerificationImages: ParentalConsentGetImagesResponse;
};