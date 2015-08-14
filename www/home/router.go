package home

import (
	"html/template"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/analytics"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/branch"
	"github.com/sprucehealth/backend/diagnosis"
	"github.com/sprucehealth/backend/libs/dispatch"
	"github.com/sprucehealth/backend/libs/golog"
	"github.com/sprucehealth/backend/libs/httputil"
	"github.com/sprucehealth/backend/libs/mux"
	"github.com/sprucehealth/backend/libs/ratelimit"
	"github.com/sprucehealth/backend/libs/sig"
	"github.com/sprucehealth/backend/libs/storage"
	"github.com/sprucehealth/backend/media"
	"github.com/sprucehealth/backend/medrecord"
	"github.com/sprucehealth/backend/www"
)

const passCookieName = "hp"

// Config is all the dependencies and settings for the home routes
type Config struct {
	DataAPI         api.DataAPI
	AuthAPI         api.AuthAPI
	SMSAPI          api.SMSAPI
	DiagnosisSvc    diagnosis.API
	WebDomain       string
	APIDomain       string
	FromSMSNumber   string
	BranchClient    branch.Client
	RateLimiters    ratelimit.KeyedRateLimiters
	Signer          *sig.Signer
	Password        string
	AnalyticsLogger analytics.Logger
	TemplateLoader  *www.TemplateLoader
	ExperimentIDs   map[string]string
	MediaStore      *media.Store
	Stores          storage.StoreMap
	Dispatcher      dispatch.Publisher
	MetricsRegistry metrics.Registry
}

// SetupRoutes configures all routes for the home website using the provided mux.
func SetupRoutes(r *mux.Router, config *Config) {
	config.TemplateLoader.MustLoadTemplate("home/base.html", "base.html", map[string]interface{}{
		"availableStates": func() string {
			// TODO: should cache this as it's mostly static
			states, err := config.DataAPI.AvailableStates()
			if err != nil {
				golog.Errorf("Failed to get list of available states: %s", err)
				// Seems like the safest fallback and doesn't seem useful to error the
				// entire page just because the list of states failed to fetch.
				return "CA, FL, NY, PA, and more"
			}
			sort.Sort(statesByAbbr(states))
			// Special cases to simplify multi-state logic
			switch len(states) {
			case 0:
				return ""
			case 1:
				return states[0].Abbreviation
			case 2:
				return states[0].Abbreviation + " and " + states[1].Abbreviation
			}
			out := make([]string, len(states)-1)
			for i, s := range states[:len(states)-1] {
				out[i] = s.Abbreviation
			}
			return strings.Join(out, ", ") + ", and " + states[len(states)-1].Abbreviation
		},
	})

	faqCtx := func() interface{} {
		return &faqContext{
			Sections: faq(config.DataAPI),
		}
	}

	r.Handle("/", newStaticHandler(r, config.TemplateLoader, "home/home.html", "Spruce", nil))
	r.Handle("/about", newStaticHandler(r, config.TemplateLoader, "home/about.html", "About | Spruce", nil))
	r.Handle("/conditions-treated", newStaticHandler(r, config.TemplateLoader, "home/conditions.html", "Conditions Treated | Spruce", nil))
	r.Handle("/contact", newStaticHandler(r, config.TemplateLoader, "home/contact.html", "Contact | Spruce", nil))
	r.Handle("/faq", newStaticHandler(r, config.TemplateLoader, "home/faq.html", "FAQ | Spruce", faqCtx))
	r.Handle("/free-visit-terms", newStaticHandler(r, config.TemplateLoader, "home/free-visit-terms.html", "Free Visit Terms & Conditions | Spruce", nil))
	r.Handle("/meet-the-doctors", newStaticHandler(r, config.TemplateLoader, "home/meet-the-doctors.html", "Meet the Doctors | Spruce", nil))
	r.Handle("/providers", newStaticHandler(r, config.TemplateLoader, "home/providers.html", "For Providers | Spruce", nil))
	r.Handle("/terms", newStaticHandler(r, config.TemplateLoader, "home/terms.html", "Terms & Conditions | Spruce", nil))
	r.Handle("/terms/2015-03-31", newStaticHandler(r, config.TemplateLoader, "home/terms-2015-03-31.html", "Terms & Conditions | Spruce", nil))
	r.Handle("/app", newStaticHandler(r, config.TemplateLoader, "home/referral.html", "Get the App | Spruce", func() interface{} {
		return &refContext{
			Title: "See a dermatologist, right from your phone.",
		}
	}))

	authFilter := func(h httputil.ContextHandler) httputil.ContextHandler {
		return www.AuthRequiredHandler(h, nil, config.AuthAPI)
	}
	authOptionalFilter := func(h httputil.ContextHandler) httputil.ContextHandler {
		return www.AuthRequiredHandler(h, h, config.AuthAPI)
	}
	r.Handle("/patient/medical-record", authFilter(newMedRecordWebDownloadHandler(config.DataAPI, config.Stores["medicalrecords"])))

	// Parental Consent
	parentalFaqCtx := func() interface{} {
		return &faqContext{
			Sections: parentalFaq(),
		}
	}
	r.Handle(`/pc/{childid:\d+}/medrecord`, authFilter(newParentalMedicalRecordHandler(config.DataAPI, &medrecord.Renderer{
		DataAPI:            config.DataAPI,
		DiagnosisSvc:       config.DiagnosisSvc,
		MediaStore:         config.MediaStore,
		APIDomain:          config.APIDomain,
		WebDomain:          config.WebDomain,
		Signer:             config.Signer,
		ExpirationDuration: time.Hour,
	})))
	config.TemplateLoader.MustLoadTemplate("home/parental-base.html", "base.html", nil)
	r.Handle("/pc/faq", newParentalLandingHandler(config.DataAPI, config.TemplateLoader, "home/parental-faq.html", "Parental Consent FAQ | Spruce", parentalFaqCtx))
	parentalConsentHandler := authOptionalFilter(newParentalConsentHandler(config.DataAPI, config.MediaStore, config.TemplateLoader))
	r.Handle(`/pc/{childid:\d+}`, parentalConsentHandler)
	r.Handle(`/pc/{childid:\d+}/{page:.*}`, parentalConsentHandler)

	// Email
	r.Handle("/e/optout", newEmailOptoutHandler(config.DataAPI, config.AuthAPI, config.Signer, config.TemplateLoader))

	// Referrals
	r.Handle("/r/{code}", newPromoClaimHandler(config.DataAPI, config.AuthAPI, config.BranchClient, config.AnalyticsLogger, config.TemplateLoader))

	// API
	apiAuthFilter := func(h httputil.ContextHandler) httputil.ContextHandler {
		return www.APIAuthRequiredHandler(h, config.AuthAPI)
	}
	r.Handle("/api/auth/sign-in", newSignInAPIHandler(config.AuthAPI))
	r.Handle("/api/auth/sign-up", newSignUpAPIHandler(config.DataAPI, config.AuthAPI))
	r.Handle("/api/forms/{form:[0-9a-z-]+}", newFormsAPIHandler(config.DataAPI))
	r.Handle("/api/textdownloadlink", newTextDownloadLinkAPIHandler(config.DataAPI, config.SMSAPI, config.FromSMSNumber, config.BranchClient, config.RateLimiters.Get("textdownloadlink")))
	r.Handle("/api/parental-consent", apiAuthFilter(newParentalConsentAPIHAndler(config.DataAPI, config.Dispatcher)))
	r.Handle("/api/parental-consent/image", apiAuthFilter(newParentalConsentImageAPIHAndler(config.DataAPI, config.Dispatcher, config.MediaStore)))

	// Analytics
	ah := newAnalyticsHandler(config.AnalyticsLogger, config.MetricsRegistry.Scope("analytics"))
	r.Handle("/api/events", ah) // For javascript originating events
	r.Handle("/a/logo.png", ah) // For remote event tracking "pixels" (e.g. email)
}

func passwordProtectFilter(pass string, templateLoader *www.TemplateLoader) func(http.Handler) http.Handler {
	tmpl := templateLoader.MustLoadTemplate("home/pass.html", "base.html", nil)
	return func(h http.Handler) http.Handler {
		return &passwordProtectHandler{
			h:    h,
			pass: pass,
			tmpl: tmpl,
		}
	}
}

type passwordProtectHandler struct {
	h    http.Handler
	pass string
	tmpl *template.Template
}

func (h *passwordProtectHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie(passCookieName)
	if err == nil {
		if c.Value == h.pass {
			h.h.ServeHTTP(w, r)
			return
		}
	}

	var errorMsg string
	if r.Method == "POST" {
		if pass := r.FormValue("Password"); pass == h.pass {
			domain := r.Host
			if i := strings.IndexByte(domain, ':'); i > 0 {
				domain = domain[:i]
			}
			http.SetCookie(w, &http.Cookie{
				Name:   passCookieName,
				Value:  pass,
				Path:   "/",
				Domain: domain,
				Secure: true,
			})
			// Redirect back to the same URL to get rid of the POST. On the next request
			// this handler should just pass through to the real handler since the cookie
			// will be set.
			http.Redirect(w, r, "", http.StatusSeeOther)
			return
		}
		errorMsg = "Invalid password."
	}
	www.TemplateResponse(w, http.StatusOK, h.tmpl, &www.BaseTemplateContext{
		Title: "Spruce",
		SubContext: &struct {
			Error string
		}{
			Error: errorMsg,
		},
	})
}

type faqContext struct {
	Sections []*faqSection
}

type faqSection struct {
	Anchor    string
	Title     string
	Questions []*faqQuestion
}

type faqQuestion struct {
	Question string
	Answer   template.HTML
}

func faq(dataAPI api.DataAPI) []*faqSection {
	var stateList string
	states, err := dataAPI.AvailableStates()
	if err != nil {
		golog.Errorf("Failed to get list of available states: %s", err)
		// Seems like the safest fallback and doesn't seem useful to error the
		// entire page just because the list of states failed to fetch.
		stateList = "California, Florida, New York, Pennsylvania, and more"
	} else {
		sort.Sort(statesByName(states))
		// Special cases to simplify multi-state logic
		switch len(states) {
		case 0:
			stateList = ""
		case 1:
			stateList = states[0].Name
		case 2:
			stateList = states[0].Name + " and " + states[1].Name
		default:
			out := make([]string, len(states)-1)
			for i, s := range states[:len(states)-1] {
				out[i] = s.Name
			}
			stateList = strings.Join(out, ", ") + ", and " + states[len(states)-1].Name
		}
	}
	stateOrList := strings.Replace(stateList, " and ", " or ", -1)

	return []*faqSection{
		{
			Title: "General",
			Questions: []*faqQuestion{
				{
					Question: "What is Spruce?",
					Answer: `
						<p>
						We connect you with board-certified US dermatologists for the professional diagnosis and treatment of a range of skin conditions including acne, anti-aging, male hair loss, rash, eczema, psoriasis, bug bites, and stings.
						</p>
						<p>Through the Spruce app, we make it easy and simple to build a relationship with a dermatologist and get medical treatment for your skin.
						</p>`,
				},
				{
					Question: "What does Spruce treat?",
					Answer:   `<p>Our dermatologists treat a broad range of skin conditions. For a full list, see our '<a href="/conditions-treated">Conditions Treated</a>' page.</p>`,
				},
				{
					Question: "How do I use Spruce?",
					Answer: `
						<p>
						Getting treated by a doctor on Spruce is easy and fast. Once you’ve downloaded the app and created an account, all you have to do is take some photos of the affected areas and answer questions about your skin and medical history.
						</p>
						<p>
						A board-certified dermatologist will then carefully review your information and create a personalized treatment plan within 24 hours. Any prescriptions in your treatment plan will be sent digitally to your preferred pharmacy.
						</p>
						<p>
						If you would like to be treated for acne and are 13 to 18 years old, your parent or guardian will need to provide consent to your treatment on Spruce before a doctor reviews your case. Once you’ve downloaded the app and created an account, you will be guided through the process of obtaining parental consent in-app. You can learn more in our section for <a href="#under18">Patients Under 18</a>.
						</p>`,
				},
				{
					Question: "Where is Spruce available?",
					Answer:   template.HTML("<p>Patients living in " + stateList + " can access Spruce. If you’re located somewhere else, don’t worry &mdash; we’ll be there soon.</p>"),
				},
				{
					Question: "Can people under 18 use Spruce?",
					Answer:   "<p>Spruce is available to patients 18 and older for treatment of our full range of dermatological concerns. Spruce is available to patients between the ages of 13 and 18 for the treatment of acne, provided that their parent has given consent for their treatment on Spruce.</p>",
				},
				{
					Question: "Are smartphone photos good enough for diagnosis?",
					Answer: `
						<p>
						The quality of cameras in phones has greatly improved over the past few years. Research shows that your dermatologist can use photos to diagnose you with a degree of accuracy that’s comparable to going to the office in person.
						</p>
						<p>To ensure that's the case, we utilize the best available photo technology that your device has to offer to ensure that doctors have the most accurate view of your skin.
						</p>`,
				},
			},
		},
		{
			Title: "Visits",
			Questions: []*faqQuestion{
				{
					Question: "How much does Spruce cost?",
					Answer:   "<p>A Spruce visit costs $40. We accept major credit and debit cards, as well as HSA and FSA cards (as long as they are branded Visa or MasterCard).</p>",
				},
				{
					Question: "What is included in a 'visit'?",
					Answer: `
						<p>
						A $40 visit includes:
						<ul>
						<li>Within 24 hours, diagnosis by a board-certified dermatologist</li>
						<li>A personalized treatment plan</li>
						<li>30 days of post-visit messaging</li>
						</ul>
						</p>`,
				},
				{
					Question: "How does messaging work?",
					Answer:   "<p>Through the app, you can message directly with your doctor and care coordinator. This allows you to ask questions about your treatment plan, when it suits you. It also allows your doctor to respond to you in a timely manner and check on your progress, where appropriate.</p>",
				},
				{
					Question: "Will I only see my doctor once?",
					Answer: `
						<p>
						Not if you don’t want to! Spruce is designed to be more than
						one-off screening and diagnosis. Most patients have an
						ongoing relationship with their care team, which includes
						completing follow-up visits when the doctor believes it’s
						necessary.
						</p>
						<p>
						A follow-up visit is much like an initial visit: you’ll be
						asked to take pictures and answer a set of questions and your
						doctor will respond with an updated treatment plan. Follow-up
						visits are never required – you will make the final decision
						about whether to proceed with one, so there won’t be any
						surprise additional charges.
						</p>`,
				},
			},
		},
		{
			Title: "Prescriptions",
			Questions: []*faqQuestion{
				{
					Question: "Will I definitely get a prescription?",
					Answer:   "<p>It’s not guaranteed, but the vast majority of treatment plans include a prescription. Your dermatologist will decide what’s needed to address your condition.</p>",
				},
				{
					Question: "Are prescriptions included in the cost of a visit?",
					Answer:   "<p>Prescriptions are not included in the cost of a visit. Just like an in-person doctor's visit, they are paid for by your insurance company or as an out-of-pocket expense. Once your care coordinator collects your coverage information, he or she can work with your doctor and/or insurance company to ensure that your treatment plan is affordable.</p>",
				},
				{
					Question: "What happens when my prescription runs out?",
					Answer: `
						<p>
						Some Spruce patients will find that their condition
						resolves after a single course of treatment. In this
						case, no refills are necessary. Where ongoing treatment
						is required, your dermatologist will make refill decisions
						in the same way as if he or she had seen you in person.
						</p>
						<p>
						For example, if it’s your first visit for a chronic
						condition like acne your prescription(s) will likely last
						two to four months, but this interval will lengthen once
						the doctor is confident the treatment is working as expected.
						When you reach the end of your refills, your dermatologist may
						request a follow-up visit prior to prescribing additional refills.
						</p>`,
				},
			},
		},
		{
			Title: "Insurance",
			Questions: []*faqQuestion{
				{
					Question: "Does Spruce work with my insurance?",
					Answer:   "<p>Your insurance cannot be used to pay for the visit but &mdash; just like an in-person doctor’s visit &mdash; it will be applied to any medications you’re prescribed when you pick them up at the pharmacy.</p>",
				},
				{
					Question: "What if I don't have insurance?",
					Answer:   `<p>You can still be treated by a dermatologist on Spruce if you don't have insurance. The cost of a visit is more affordable than what you would pay for an in-person visit without coverage. Prescriptions are not included in the $40, but both your care coordinator and doctor will be notified of your insurance status and will work together to find effective, less expensive medications.</p>`,
				},
			},
		},
		{
			Title: "Doctors",
			Questions: []*faqQuestion{
				{
					Question: "Who are the doctors that will be treating me?",
					Answer: `
						<p>
						Spruce connects you to top dermatologists from across the
						country. In addition to being board-certified, these doctors
						are from accredited medical schools and have undergone a
						rigorous interview process (including full background checks).
						</p>
						<p>
						All of the doctors within the Spruce network currently have
						in-person practices too, meaning you can expect the same level
						of clinical experience that you would receive in a traditional
						visit. You can learn more about our doctors on our 'Meet the
						Doctors' page and can also view their profiles within the app.
						</p>`,
				},
				{
					Question: "What if I don’t like my dermatologist?",
					Answer:   "<p>Let us know and we’ll be happy to help you find a physician you’re excited to work with. You can submit a request within the app via ‘Care Team’ in the Menu, or by emailing the Spruce Customer Support team directly.</p>",
				},
			},
		},
		{
			Title: "Information & Privacy",
			Questions: []*faqQuestion{
				{
					Question: "How can I keep my primary care physician in the loop?",
					Answer:   "<p>Spruce enables you to download your care record, which you can then print or share with your PCP (or any other doctor). You can do this by clicking on the ‘Care Record’ tab, then ‘Export Care Record’.</p>",
				},
				{
					Question: "How will you keep the things I share private?",
					Answer:   "<p>Spruce is a HIPAA-compliant service, and all personal and medical information is protected according to the highest industry standards. We incorporate multiple layers of security, and encrypt your data both “over the wire” (when transmitted to and from your device) and “at rest” in the database (which itself is protected by strict access controls and physical measures).</p>",
				},
			},
		},
		{
			Anchor: "under18",
			Title:  "Patients Under 18",
			Questions: []*faqQuestion{
				{
					Question: "Can I be treated by a dermatologist on Spruce?",
					Answer: template.HTML(`
						<p>You can be treated on Spruce for acne if:</p>
						<ul>
						<li>You are between the ages of 13 and 18;</li>
						<li>You live in ` + stateOrList + `; and</li>
						<li>A parent provides consent for your treatment.</li>
						</ul>`),
				},
				{
					Question: "What does Spruce treat for patients under 18?",
					Answer:   "<p>At this time, patients between the ages of 13 and 18 can only be treated for acne on Spruce.</p>",
				},
				{
					Question: "Do I need Spruce for my acne?",
					Answer:   `<p>If you have been using over-the-counter products to treat your acne without results, it’s time to consult a dermatologist. According to the American Academy of Dermatology, 99% of acne cases are treatable. A dermatologist can diagnose the type of acne you have and prescribe an appropriate skin care regimen.</p>`,
				},
				{
					Question: "How do I obtain consent from my parent or guardian?",
					Answer:   `<p>Once you have downloaded the app, created an account, and submitted your information, you will be guided through the process of obtaining consent from your parent or guardian. </p>`,
				},
				{
					Question: "I am a parent - how can I arrange for my teen to be treated on Spruce? ",
					Answer: `
						<p>Your child can be treated on Spruce by downloading the Spruce app on their smartphone, creating an account, taking several photos, and answering questions about their skin and medical history. If you’d like, you can walk through this with them, but supervising the entire process is not necessary.</p>
						<p>Before a doctor reviews the case, your child will need to prompt you to provide parental consent. As their parent, you will be guided through the consent process, which can be completed on your phone or desktop and which will also present you with the option to pay for the cost of the visit on your child’s behalf.</p>
					`,
				},
				{
					Question: "I am a parent - how do I stay informed about my child’s treatment on Spruce?",
					Answer: `
						<p>As part of providing your consent, you’ll create a Spruce parent account, which you can use to log in and view your child’s care record. The care record includes your child's visit information, treatment plan, and messages with their care team.</p>
						<p>We’ll notify you via email of any major updates in your child’s case, and you can always contact us with any questions at <a href="mailto:support@sprucehealth.com">support@sprucehealth.com</a>.</p>
					`,
				},
			},
		},
	}
}

func parentalFaq() []*faqSection {
	return []*faqSection{
		{
			Title: "General",
			Questions: []*faqQuestion{
				{
					Question: "Why did I get sent this? How did you get my details?",
					Answer: `
						<p>
						You were sent this because your child (or legal dependent) found Spruce and wanted to start a visit with a board-certified dermatologist to treat their acne in an affordable and convenient way. Before a doctor can treat their case, we would like to obtain parental consent, like any in-person dermatologist would.
						</p>`,
				},
				{
					Question: "How did my child find out about Spruce?",
					Answer: `
						<p>
						Your child may have discovered Spruce through a variety of means.  Dermatologists have been treating patients on Spruce since 2014 for acne, psoriasis, eczema, rash, bug bites, and a range of other skin conditions. Your child may have heard about Spruce through one of our existing patients. Additionally, Spruce has been featured in a variety of high-quality news outlets and publications and actively uses digital advertising.
						</p>`,
				},
				{
					Question: "Does my child really need to see a dermatologist for acne?",
					Answer: `
						<p>
						If your child has been using over-the-counter products to treat their acne without results, it’s time to consult a dermatologist. According to the American Academy of Dermatology, 99&#37; of acne cases are treatable. A dermatologist can diagnose the type of acne your child has and prescribe an appropriate skin care regimen.
						</p>`,
				},
				{
					Question: "Are smartphone photos good enough for diagnosis? ",
					Answer: `<p>
						The quality of cameras in phones has greatly improved over the past few years. Research shows that your dermatologist can use photos to diagnose you with a degree of accuracy that’s comparable to going to the office in person.
						</p>
						<p>
						To ensure that's the case, we utilize the best available photo technology that your device has to offer so that your doctor on Spruce will have the most accurate view of your skin.
						</p>`,
				},
			},
		},
		{
			Title: "Parental Consent",
			Questions: []*faqQuestion{
				{
					Question: "How do I provide my consent? ",
					Answer: `
						<p>
						You can provide consent by confirming your identity with Spruce and creating an account. This will allow you to keep tabs on your child’s care and will act as your acknowledgement of Spruce’s terms, privacy policy, and consent to the use of telehealth.
						</p>
						<p>
						If you do not wish to consent to your child’s treatment on Spruce, talk to your child and let them know that they won’t be able to continue their visit or be treated by a dermatologist on Spruce.
						</p>`,
				},
				{
					Question: "Why do you need a photo of my ID?",
					Answer: `
						<p>
						We are committed to protecting the safety of children on Spruce. To do this, we need to be confident that the adults responsible for them are of age to consent to treatment. Your photo ID will only be used for this purpose.
						</p>`,
				},
				{
					Question: "How do I know what’s happening between my child and their doctor? ",
					Answer: `
						<p>
						You’ll have access to your child’s care record on Spruce which includes their visit information, treatment plan, and messages with their care team.
						</p>`,
				},
			},
		},
		{
			Title: "Visits",
			Questions: []*faqQuestion{
				{
					Question: "Are prescription costs included in the price of a visit?",
					Answer: `
						<p>
						Prescription costs are not included in the price of a visit. Just like an in-person doctor's visit, medications are paid for by your insurance company or as an out-of-pocket expense. Your care coordinator can work with your child’s doctor and/or their covering insurance company to ensure that your treatment plan is affordable.
						</p>`,
				},
				{
					Question: "Can I use my insurance to pay for the visit?",
					Answer: `
						<p>
						Your insurance cannot be used to pay for the visit but &mdash; just like an in-person doctor’s visit &mdash; it will be applied to any medications you’re prescribed when you pick them up at the pharmacy.
						</p>`,
				},
			},
		},
		{
			Title: "Information & Privacy",
			Questions: []*faqQuestion{
				{
					Question: "Is this secure? How do you keep my child’s information private?",
					Answer: `
						<p>
						Spruce is a HIPAA-compliant service, and all personal and medical information is protected according to the highest industry standards. We incorporate multiple layers of security, and encrypt your data both "over the wire" (when transmitted to and from your device) and "at rest" in the database (which itself is protected by strict access controls and physical measures).
						</p>`,
				},
				{
					Question: "How can I keep my primary care physician in the loop?",
					Answer: `
						<p>
						Spruce enables you to download your care record, which you can then print or share with your PCP (or any other doctor). You can do this by clicking on the 'Care Record' tab, then 'Export Care Record'.
						</p>`,
				},
			},
		},
	}
}
