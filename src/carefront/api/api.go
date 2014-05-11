package api

import (
	"carefront/libs/pharmacy"
	"errors"
	"net/http"
	"reflect"
	"time"

	"carefront/common"
)

const (
	EN_LANGUAGE_ID                 = 1
	DOCTOR_ROLE                    = "DOCTOR"
	PRIMARY_DOCTOR_STATUS          = "PRIMARY"
	PATIENT_ROLE                   = "PATIENT"
	REVIEW_PURPOSE                 = "REVIEW"
	CONDITION_INTAKE_PURPOSE       = "CONDITION_INTAKE"
	DIAGNOSE_PURPOSE               = "DIAGNOSE"
	FOLLOW_UP_WEEK                 = "week"
	FOLLOW_UP_DAY                  = "day"
	FOLLOW_UP_MONTH                = "month"
	CASE_STATUS_OPEN               = "OPEN"
	CASE_STATUS_SUBMITTED          = "SUBMITTED"
	CASE_STATUS_REVIEWING          = "REVIEWING"
	CASE_STATUS_CLOSED             = "CLOSED"
	CASE_STATUS_TRIAGED            = "TRIAGED"
	CASE_STATUS_TREATED            = "TREATED"
	CASE_STATUS_PHOTOS_REJECTED    = "PHOTOS_REJECTED"
	REFILL_REQUEST_STATUS_PENDING  = "PENDING"
	REFILL_REQUEST_STATUS_SENDING  = "SENDING"
	REFILL_REQUEST_STATUS_APPROVED = "APPROVED"
	REFILL_REQUEST_STATUS_DENIED   = "DENIED"
	PATIENT_UNLINKED               = "UNLINKED"
	PATIENT_REGISTERED             = "REGISTERED"
	DOCTOR_REGISTERED              = "REGISTERED"
	HIPAA_AUTH                     = "hipaa"
	CONSENT_AUTH                   = "consent"
	PHONE_HOME                     = "Home"
	PHONE_WORK                     = "Work"
	PHONE_CELL                     = "Cell"
)

var (
	NoRowsError                 = errors.New("No rows exist")
	NoElligibileProviderInState = errors.New("There are no providers elligible in the state the patient resides")
	NoDiagnosisResponseErr      = errors.New("No diagnosis response exists to the question queried tag queried with")
)

type PotentialAnswerInfo struct {
	PotentialAnswerId int64
	AnswerType        string
	Answer            string
	AnswerSummary     string
	AnswerTag         string
	Ordering          int64
	ToAlert           bool
}

type PatientAPI interface {
	GetPatientFromId(patientId int64) (patient *common.Patient, err error)
	GetPatientFromAccountId(accountId int64) (patient *common.Patient, err error)
	GetPatientFromErxPatientId(erxPatientId int64) (*common.Patient, error)
	GetPatientFromRefillRequestId(refillRequestId int64) (*common.Patient, error)
	GetPatientFromTreatmentId(treatmentId int64) (*common.Patient, error)
	GetPatientFromUnlinkedDNTFTreatment(unlinkedDNTFTreatmentId int64) (*common.Patient, error)
	RegisterPatient(patient *common.Patient) error
	UpdateTopLevelPatientInformation(patient *common.Patient) error
	UpdatePatientInformation(patient *common.Patient, updateFromDoctor bool) error
	CreateUnlinkedPatientFromRefillRequest(patient *common.Patient) error
	UpdatePatientWithERxPatientId(patientId, erxPatientId int64) error
	GetPatientIdFromAccountId(accountId int64) (int64, error)
	CreateCareTeamForPatient(patientId int64) (careTeam *common.PatientCareProviderGroup, err error)
	CreateCareTeamForPatientWithPrimaryDoctor(patientId, doctorId int64) (careTeam *common.PatientCareProviderGroup, err error)
	GetCareTeamForPatient(patientId int64) (careTeam *common.PatientCareProviderGroup, err error)
	CheckCareProvidingElligibility(shortState string, healthConditionId int64) (doctorId int64, err error)

	UpdatePatientAddress(patientId int64, addressLine1, addressLine2, city, state, zipCode, addressType string) error
	UpdatePatientPharmacy(patientId int64, pharmacyDetails *pharmacy.PharmacyData) error
	TrackPatientAgreements(patientId int64, agreements map[string]bool) error
	GetPatientFromPatientVisitId(patientVisitId int64) (patient *common.Patient, err error)
	GetPatientFromTreatmentPlanId(treatmentPlanId int64) (patient *common.Patient, err error)
	GetPatientsForIds(patientIds []int64) ([]*common.Patient, error)
	GetPharmacySelectionForPatients(patientIds []int64) ([]*pharmacy.PharmacyData, error)
	GetPharmacyBasedOnReferenceIdAndSource(pharmacyid, pharmacySource string) (*pharmacy.PharmacyData, error)
	GetPharmacyFromId(pharmacyLocalId int64) (*pharmacy.PharmacyData, error)
	AddPharmacy(pharmacyDetails *pharmacy.PharmacyData) error
	UpdatePatientWithPaymentCustomerId(patientId int64, paymentCustomerId string) error
	CreatePendingTask(workType, status string, itemId int64) (int64, error)
	DeletePendingTask(pendingTaskId int64) error
	AddCardAndMakeDefaultForPatient(patientId int64, card *common.Card) error
	MarkCardInactiveForPatient(patientId int64, card *common.Card) error
	DeleteCardForPatient(patientId int64, card *common.Card) error
	MakeLatestCardDefaultForPatient(patientId int64) (*common.Card, error)
	MakeCardDefaultForPatient(patientId int64, card *common.Card) error
	GetCardsForPatient(patientId int64) ([]*common.Card, error)
	GetCardFromId(cardId int64) (*common.Card, error)
	UpdateDefaultAddressForPatient(patientId int64, address *common.Address) error
	DeleteAddress(addressId int64) error
	GetFullNameForState(state string) (string, error)
}

type PatientVisitAPI interface {
	GetActivePatientVisitIdForHealthCondition(patientId, healthConditionId int64) (int64, error)
	GetLastCreatedPatientVisitIdForPatient(patientId int64) (int64, error)
	GetPatientIdFromPatientVisitId(patientVisitId int64) (int64, error)
	GetLatestSubmittedPatientVisit() (*common.PatientVisit, error)
	GetPatientVisitIdFromTreatmentPlanId(treatmentPlanId int64) (int64, error)
	GetLatestClosedPatientVisitForPatient(patientId int64) (*common.PatientVisit, error)
	GetPatientVisitFromId(patientVisitId int64) (patientVisit *common.PatientVisit, err error)
	CreateNewPatientVisit(patientId, healthConditionId, layoutVersionId int64) (int64, error)
	StartNewTreatmentPlanForPatientVisit(patientId, patientVisitId, doctorId, favoriteTreatmentPlanId int64) (int64, error)
	GetAbbreviatedTreatmentPlanForPatientVisit(doctorId, patientVisitId int64) (*common.DoctorTreatmentPlan, error)
	GetActiveTreatmentPlanForPatientVisit(doctorId, patientVisitId int64) (int64, error)
	UpdatePatientVisitStatus(patientVisitId int64, message, event string) error
	GetMessageForPatientVisitStatus(patientVisitId int64) (message string, err error)
	ClosePatientVisit(patientVisitId, treatmentPlanId int64, event, message string) error
	SubmitPatientVisitWithId(patientVisitId int64) error
	UpdateFollowUpTimeForPatientVisit(treatmentPlanId, doctorId, currentTimeSinceEpoch, followUpValue int64, followUpUnit string) error
	GetFollowUpTimeForTreatmentPlan(treatmentPlanId int64) (followUp *common.FollowUp, err error)
	GetDiagnosisResponseToQuestionWithTag(questionTag string, doctorId, patientVisitId int64) ([]*common.AnswerIntake, error)
	AddDiagnosisSummaryForTreatmentPlan(summary string, treatmentPlanId, doctorId int64) error
	GetDiagnosisSummaryForTreatmentPlan(treatmentPlanId int64) (*common.DiagnosisSummary, error)
	AddOrUpdateDiagnosisSummaryForTreatmentPlan(summary string, treatmentPlanId, doctorId int64, isUpdatedByDoctor bool) error
	DeactivatePreviousDiagnosisForPatientVisit(treatmentPlanId int64, doctorId int64) error
	RecordDoctorAssignmentToPatientVisit(patientVisitId, doctorId int64) error
	GetDoctorAssignedToPatientVisit(patientVisitId int64) (doctor *common.Doctor, err error)
	GetAdvicePointsForTreatmentPlan(treatmentPlanId int64) (advicePoints []*common.DoctorInstructionItem, err error)
	CreateAdviceForPatientVisit(advicePoints []*common.DoctorInstructionItem, treatmentPlanId int64) error
	CreateRegimenPlanForPatientVisit(regimenPlan *common.RegimenPlan) error
	GetRegimenPlanForTreatmentPlan(treatmentPlanId int64) (regimenPlan *common.RegimenPlan, err error)
	AddTreatmentsForPatientVisit(treatments []*common.Treatment, doctorId, treatmentPlanId, patientId int64) error
	GetTreatmentsBasedOnTreatmentPlanId(patientVisitId, treatmentPlanId int64) ([]*common.Treatment, error)
	GetTreatmentBasedOnPrescriptionId(erxId int64) (*common.Treatment, error)
	GetTreatmentsForPatient(patientId int64) ([]*common.Treatment, error)
	GetTreatmentFromId(treatmentId int64) (*common.Treatment, error)
	UpdateTreatmentWithPharmacyAndErxId(treatments []*common.Treatment, pharmacySentTo *pharmacy.PharmacyData, doctorId int64) error
	AddErxStatusEvent(treatments []*common.Treatment, prescriptionStatus common.StatusEvent) error
	GetPrescriptionStatusEventsForPatient(patientId int64) ([]common.StatusEvent, error)
	GetPrescriptionStatusEventsForTreatment(treatmentId int64) ([]common.StatusEvent, error)
}

type RefillRequestDenialReason struct {
	Id           int64  `json:"id,string"`
	DenialCode   string `json:"denial_code"`
	DenialReason string `json:"denial_reason"`
}

type PrescriptionsAPI interface {
	GetPendingRefillRequestStatusEventsForClinic() ([]common.StatusEvent, error)
	GetApprovedOrDeniedRefillRequestsForPatient(patientId int64) ([]common.StatusEvent, error)
	GetRefillStatusEventsForRefillRequest(refillRequestId int64) ([]common.StatusEvent, error)
	CreateRefillRequest(*common.RefillRequestItem) error
	AddRefillRequestStatusEvent(refillRequestStatus common.StatusEvent) error
	GetRefillRequestFromId(refillRequestId int64) (*common.RefillRequestItem, error)
	GetRefillRequestsForPatient(patientId int64) ([]*common.RefillRequestItem, error)
	GetRefillRequestDenialReasons() ([]*RefillRequestDenialReason, error)
	MarkRefillRequestAsApproved(prescriptionId, approvedRefillCount, rxRefillRequestId int64, comments string) error
	MarkRefillRequestAsDenied(prescriptionId, denialReasonId, rxRefillRequestId int64, comments string) error
	LinkRequestedPrescriptionToOriginalTreatment(requestedTreatment *common.Treatment, patient *common.Patient) error
	AddUnlinkedTreatmentInEventOfDNTF(treatment *common.Treatment, refillRequestId int64) error
	GetUnlinkedDNTFTreatment(treatmentId int64) (*common.Treatment, error)
	GetUnlinkedDNTFTreatmentsForPatient(patientId int64) ([]*common.Treatment, error)
	AddTreatmentToTreatmentPlanInEventOfDNTF(treatment *common.Treatment, refillRequestId int64) error
	UpdateUnlinkedDNTFTreatmentWithPharmacyAndErxId(treatment *common.Treatment, pharmacySentTo *pharmacy.PharmacyData, doctorId int64) error
	AddErxStatusEventForDNTFTreatment(statusEvent common.StatusEvent) error
	GetErxStatusEventsForDNTFTreatment(treatmentId int64) ([]common.StatusEvent, error)
	GetErxStatusEventsForDNTFTreatmentBasedOnPatientId(patientId int64) ([]common.StatusEvent, error)
}

type DrugAPI interface {
	DoesDrugDetailsExist(ndc string) (bool, error)
	DrugDetails(ndc string) (*common.DrugDetails, error)
	SetDrugDetails(ndcToDrugDetails map[string]*common.DrugDetails) error
}

type DoctorAPI interface {
	RegisterDoctor(doctor *common.Doctor) (int64, error)
	GetDoctorFromId(doctorId int64) (doctor *common.Doctor, err error)
	GetDoctorFromAccountId(accountId int64) (doctor *common.Doctor, err error)
	GetDoctorFromDoseSpotClinicianId(clincianId int64) (doctor *common.Doctor, err error)
	GetDoctorIdFromAccountId(accountId int64) (int64, error)
	GetRegimenStepsForDoctor(doctorId int64) (regimenSteps []*common.DoctorInstructionItem, err error)
	GetRegimenStepForDoctor(regimenStepId, doctorId int64) (*common.DoctorInstructionItem, error)
	AddRegimenStepForDoctor(regimenStep *common.DoctorInstructionItem, doctorId int64) error
	UpdateRegimenStepForDoctor(regimenStep *common.DoctorInstructionItem, doctorId int64) error
	MarkRegimenStepToBeDeleted(regimenStep *common.DoctorInstructionItem, doctorId int64) error
	MarkRegimenStepsToBeDeleted(regimenSteps []*common.DoctorInstructionItem, doctorId int64) error
	GetAdvicePointsForDoctor(doctorId int64) (advicePoints []*common.DoctorInstructionItem, err error)
	GetAdvicePointForDoctor(advicePointId, doctorId int64) (*common.DoctorInstructionItem, error)
	AddAdvicePointForDoctor(advicePoint *common.DoctorInstructionItem, doctorId int64) error
	UpdateAdvicePointForDoctor(advicePoint *common.DoctorInstructionItem, doctorId int64) error
	MarkAdvicePointToBeDeleted(advicePoint *common.DoctorInstructionItem, doctorId int64) error
	MarkAdvicePointsToBeDeleted(advicePoints []*common.DoctorInstructionItem, doctorId int64) error
	MarkPatientVisitAsOngoingInDoctorQueue(doctorId, patientVisitId int64) error
	GetPendingItemsInDoctorQueue(doctorId int64) (doctorQueue []*DoctorQueueItem, err error)
	GetCompletedItemsInDoctorQueue(doctorId int64) (doctorQueue []*DoctorQueueItem, err error)
	GetMedicationDispenseUnits(languageId int64) (dispenseUnitIds []int64, dispenseUnits []string, err error)
	GetDrugInstructionsForDoctor(drugName, drugForm, drugRoute string, doctorId int64) (drugInstructions []*common.DoctorInstructionItem, err error)
	AddOrUpdateDrugInstructionForDoctor(drugName, drugForm, drugRoute string, drugInstructionToAdd *common.DoctorInstructionItem, doctorId int64) error
	DeleteDrugInstructionForDoctor(drugInstructionToDelete *common.DoctorInstructionItem, doctorId int64) error
	AddDrugInstructionsToTreatment(drugName, drugForm, drugRoute string, drugInstructions []*common.DoctorInstructionItem, treatmentId int64, doctorId int64) error
	AddTreatmentTemplates(treatments []*common.DoctorTreatmentTemplate, doctorId int64) error
	GetTreatmentTemplates(doctorId int64) ([]*common.DoctorTreatmentTemplate, error)
	DeleteTreatmentTemplates(doctorTreatmentTemplates []*common.DoctorTreatmentTemplate, doctorId int64) error
	InsertItemIntoDoctorQueue(doctorQueueItem DoctorQueueItem) error
	ReplaceItemInDoctorQueue(doctorQueueItem DoctorQueueItem, currentState string) error
	DeleteItemFromDoctorQueue(doctorQueueItem DoctorQueueItem) error
	MarkGenerationOfTreatmentPlanInVisitQueue(doctorId, patientVisitId, treatmentPlanId int64, currentState, updatedState string) error
}

type FavoriteTreatmentPlanAPI interface {
	CreateOrUpdateFavoriteTreatmentPlan(favoriteTreatmentPlan *common.FavoriteTreatmentPlan) error
	GetFavoriteTreatmentPlansForDoctor(doctorId int64) ([]*common.FavoriteTreatmentPlan, error)
	GetFavoriteTreatmentPlan(favoriteTreatmentPlanId int64) (*common.FavoriteTreatmentPlan, error)
	DeleteFavoriteTreatmentPlan(favoriteTreatmentPlanId int64) error
	GetTreatmentsInFavoriteTreatmentPlan(favoriteTreatmentPlanId int64) ([]*common.Treatment, error)
	GetRegimenPlanInFavoriteTreatmentPlan(favoriteTreatmentPlanId int64) (*common.RegimenPlan, error)
	GetAdviceInFavoriteTreatmentPlan(favoriteTreatmentPlanId int64) (*common.Advice, error)
	DeleteFavoriteTreatmentPlanMapping(treatmentPlanId, favoriteTreatmentPlanId int64) error
}

type IntakeAPI interface {
	GetPatientAnswersForQuestionsInGlobalSections(questionIds []int64, patientId int64) (map[int64][]*common.AnswerIntake, error)
	GetPatientAnswersForQuestionsBasedOnQuestionIds(questionIds []int64, roleId int64, patientVisitId int64) (map[int64][]*common.AnswerIntake, error)
	GetDoctorAnswersForQuestionsInDiagnosisLayout(questionIds []int64, roleId int64, patientVisitId int64) (map[int64][]*common.AnswerIntake, error)
	StoreAnswersForQuestion(role string, roleId, patientVisitId, layoutVersionId int64, answersToStorePerQuestion map[int64][]*common.AnswerIntake) error
	CreatePhotoAnswerForQuestionRecord(role string, roleId, questionId, patientVisitId, potentialAnswerId, layoutVersionId int64) (patientInfoIntakeId int64, err error)
	UpdatePhotoAnswerRecordWithObjectStorageId(patientInfoIntakeId, objectStorageId int64) error
	MakeCurrentPhotoAnswerInactive(role string, roleId, questionId, patientVisitId, potentialAnswerId, layoutVersionId int64) error
	RejectPatientVisitPhotos(patientVisitId int64) error
}

type IntakeLayoutAPI interface {
	GetQuestionType(questionId int64) (questionType string, err error)
	GetActiveLayoutInfoForHealthCondition(healthConditionTag, role, purpose string) (bucket, key, region string, err error)
	GetStorageInfoOfCurrentActivePatientLayout(languageId, healthConditionId int64) (bucket, key, region string, layoutVersionId int64, err error)
	GetStorageInfoOfCurrentActiveDoctorLayout(healthConditionId int64) (bucket, storage, region string, layoutVersionId int64, err error)
	GetStorageInfoOfActiveDoctorDiagnosisLayout(healthConditionId int64) (bucket, storage, region string, layoutVersionId int64, err error)
	GetLayoutVersionIdForPatientVisit(patientVisitId int64) (layoutVersionId int64, err error)
	GetStorageInfoForClientLayout(layoutVersionId, languageId int64) (bucket, key, region string, err error)
	MarkNewLayoutVersionAsCreating(objectId int64, syntaxVersion int64, healthConditionId int64, role, purpose, comment string) (int64, error)
	MarkNewPatientLayoutVersionAsCreating(objectId int64, languageId int64, layoutVersionId int64, healthConditionId int64) (int64, error)
	UpdatePatientActiveLayouts(layoutId int64, clientLayoutIds []int64, healthConditionId int64) error
	MarkNewDoctorLayoutAsCreating(objectId int64, layoutVersionId int64, healthConditionId int64) (int64, error)
	UpdateDoctorActiveLayouts(layoutId, doctorLayoutId, healthConditionId int64, purpose string) error
	GetGlobalSectionIds() ([]int64, error)
	GetSectionIdsForHealthCondition(healthConditionId int64) ([]int64, error)
	GetHealthConditionInfo(healthConditionTag string) (int64, error)
	GetSectionInfo(sectionTag string, languageId int64) (id int64, title string, err error)
	GetQuestionInfo(questionTag string, languageId int64) (*common.QuestionInfo, error)
	GetQuestionInfoForTags(questionTags []string, languageId int64) ([]*common.QuestionInfo, error)
	GetAnswerInfo(questionId int64, languageId int64) (answerInfos []PotentialAnswerInfo, err error)
	GetAnswerInfoForTags(answerTags []string, languageId int64) ([]PotentialAnswerInfo, error)
	GetTipSectionInfo(tipSectionTag string, languageId int64) (id int64, tipSectionTitle string, tipSectionSubtext string, err error)
	GetTipInfo(tipTag string, languageId int64) (id int64, tip string, err error)
	GetSupportedLanguages() (languagesSupported []string, languagesSupportedIds []int64, err error)
}

type ObjectStorageDBAPI interface {
	CreateNewUploadCloudObjectRecord(bucket, key, region string) (int64, error)
	UpdateCloudObjectRecordToSayCompleted(id int64) error
}

type HomeAPI interface {
	// Notifications
	DeletePatientNotifications(ids []int64) error
	DeletePatientNotificationByUID(patientId int64, uid string) error
	GetNotificationsForPatient(patientId int64, typeMap map[string]reflect.Type) (notes []*common.Notification, badNotes []*common.Notification, err error)
	InsertPatientNotification(patientId int64, note *common.Notification) (int64, error)
	// Health Log
	GetHealthLogForPatient(patientId int64, typeMap map[string]reflect.Type) (items []*common.HealthLogItem, badItems []*common.HealthLogItem, err error)
	InsertOrUpdatePatientHealthLogItem(patientId int64, item *common.HealthLogItem) (int64, error)
}

type PeopleAPI interface {
	GetPeople(ids []int64) (map[int64]*common.Person, error)
	GetPersonIdByRole(roleType string, roleId int64) (int64, error)
}

type MessageAPI interface {
	GetConversationParticipantIds(conversationId int64) ([]int64, error)
	GetConversationTopics() ([]*common.ConversationTopic, error)
	AddConversationTopic(title string, ordinal int, active bool) (int64, error)
	GetConversationsWithParticipants(ids []int64) ([]*common.Conversation, map[int64]*common.Person, error)
	GetConversation(id int64) (*common.Conversation, error)
	MarkConversationAsRead(id int64) error
	CreateConversation(fromId, toId, topicId int64, message string, attachments []*common.ConversationAttachment) (int64, error)
	ReplyToConversation(conversationId, fromId int64, message string, attachments []*common.ConversationAttachment) (int64, error)
}

type PhotoAPI interface {
	AddPhoto(uploaderId int64, url, mimetype string) (int64, error)
	GetPhoto(photoID int64) (*common.Photo, error)
	ClaimPhoto(photoId int64, claimerType string, claimerId int64) error
}

type DataAPI interface {
	PatientAPI
	DoctorAPI
	PatientVisitAPI
	IntakeLayoutAPI
	ObjectStorageDBAPI
	IntakeAPI
	PrescriptionsAPI
	DrugAPI
	HomeAPI
	PeopleAPI
	MessageAPI
	PhotoAPI
	FavoriteTreatmentPlanAPI
}

type CloudStorageAPI interface {
	GetObjectAtLocation(bucket, key, region string) (rawData []byte, responseHeader http.Header, err error)
	GetSignedUrlForObjectAtLocation(bucket, key, region string, duration time.Time) (url string, err error)
	DeleteObjectAtLocation(bucket, key, region string) error
	PutObjectToLocation(bucket, key, region, contentType string, rawData []byte, duration time.Time, dataApi DataAPI) (int64, string, error)
}
