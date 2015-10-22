package attribution

const (
	// AKCareProviderID is the key that should map to the care_provider_id in attribution data sets
	AKCareProviderID = "care_provider_id"

	// AKPromotionCode is the key that should map to the attribution promo_code
	AKPromotionCode = "promo_code"

	// AKPathwayTag is the key that should map to any pathway tag associated with the attribution data
	AKPathwayTag = "pathway_tag"

	// AKSprucePatient is the key to indicate that the patient coming in via the attribution is indeed
	// a spruce patient and not a practice extension patient.
	AKSprucePatient = "is_spruce_patient"
)
