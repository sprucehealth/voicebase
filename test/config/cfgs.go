package config

import (
	"time"

	"github.com/sprucehealth/backend/libs/cfg"
)

// GlobalFirstVisitFreeDisabled is a disabled test config
var GlobalFirstVisitFreeDisabled = &cfg.ValueDef{
	Name:        "Global.First.Visit.Free.Enabled",
	Description: "A value that represents if the first visit should be free for all patients.",
	Type:        cfg.ValueTypeBool,
	Default:     false,
}

// GlobalFirstVisitFreeDisabled is an enabled test config
var GlobalFirstVisitFreeEnabled = &cfg.ValueDef{
	Name:        "Global.First.Visit.Free.Enabled",
	Description: "A value that represents if the first visit should be free for all patients.",
	Type:        cfg.ValueTypeBool,
	Default:     true,
}

// WelcomeEmailEnabled allows a suer to enable or disable the welcome email campaigne
var WelcomeEmailEnabled = &cfg.ValueDef{
	Name:        "Email.Campaign.Welcome.Enabled",
	Description: "Enable or disable the welcome email.",
	Type:        cfg.ValueTypeBool,
	Default:     true,
}

// WelcomeEmailDisabled allows a suer to enable or disable the welcome email campaigne
var WelcomeEmailDisabled = &cfg.ValueDef{
	Name:        "Email.Campaign.Welcome.Enabled",
	Description: "Enable or disable the welcome email.",
	Type:        cfg.ValueTypeBool,
	Default:     false,
}

// MinorTreatmentPlanIssuedEmailEnabled allows a user to enable or disable the email notifying the parent account when a minor attached to their account has been issued a treatment plan.
var MinorTreatmentPlanIssuedEmailEnabled = &cfg.ValueDef{
	Name:        "Email.Campaign.Minor.Treatment.Plan.Issued.Enabled",
	Description: "Enable or disable the email notifying the parent account when a minor attached to their account has been issued a treatment plan.",
	Type:        cfg.ValueTypeBool,
	Default:     true,
}

// MinorTreatmentPlanIssuedEmailDisabled allows a user to enable or disable the email notifying the parent account when a minor attached to their account has been issued a treatment plan.
var MinorTreatmentPlanIssuedEmailDisabled = &cfg.ValueDef{
	Name:        "Email.Campaign.Minor.Treatment.Plan.Issued.Enabled",
	Description: "Enable or disable the email notifying the parent account when a minor attached to their account has been issued a treatment plan.",
	Type:        cfg.ValueTypeBool,
	Default:     false,
}

// MinorTriagedEmailEnabledDef allows a user to enable or disable the email notifying the parent account when a minor attached to their account has been triaged.
var MinorTriagedEmailEnabled = &cfg.ValueDef{
	Name:        "Email.Campaign.Minor.Triaged.Enabled",
	Description: "Enable or disable the email notifying the parent account when a minor attached to their account has been triaged.",
	Type:        cfg.ValueTypeBool,
	Default:     true,
}

// MinorTriagedEmailDisabled allows a user to enable or disable the email notifying the parent account when a minor attached to their account has been triaged.
var MinorTriagedEmailDisabled = &cfg.ValueDef{
	Name:        "Email.Campaign.Minor.Triaged.Enabled",
	Description: "Enable or disable the email notifying the parent account when a minor attached to their account has been triaged.",
	Type:        cfg.ValueTypeBool,
	Default:     false,
}

// ParentWelcomeEmailEnabled allows a user to enable or disable the email welcoming parents post consent.
var ParentWelcomeEmailEnabled = &cfg.ValueDef{
	Name:        "Email.Campaign.Parent.Welcome.Enabled",
	Description: "Enable or disable the email welcoming parents after consenting.",
	Type:        cfg.ValueTypeBool,
	Default:     true,
}

// ParentWelcomeEmailDisabled allows a user to enable or disable the email welcoming parents post consent.
var ParentWelcomeEmailDisabled = &cfg.ValueDef{
	Name:        "Email.Campaign.Parent.Welcome.Enabled",
	Description: "Enable or disable the email welcoming parents after consenting.",
	Type:        cfg.ValueTypeBool,
	Default:     false,
}

// AbandonedVisitThreshold indicates the duration of the visit in the OPEN state
// after which it is considered abandoned.
var AbandonedVisitThreshold = &cfg.ValueDef{
	Name:        "Email.Campaign.AbandonedVisit.After",
	Description: "Age of an open visit after which it's considered abandoned. Set to 0 to disable.",
	Type:        cfg.ValueTypeDuration,
	Default:     time.Hour * 24 * 7,
}
