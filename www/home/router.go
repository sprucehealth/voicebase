package home

import (
	"html/template"
	"net/http"
	"strings"

	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/sprucehealth/backend/Godeps/_workspace/src/github.com/samuel/go-metrics/metrics"
	"github.com/sprucehealth/backend/analytics"
	"github.com/sprucehealth/backend/api"
	"github.com/sprucehealth/backend/branch"
	"github.com/sprucehealth/backend/libs/ratelimit"
	"github.com/sprucehealth/backend/libs/sig"
	"github.com/sprucehealth/backend/www"
)

const passCookieName = "hp"

func SetupRoutes(
	r *mux.Router,
	dataAPI api.DataAPI,
	authAPI api.AuthAPI,
	smsAPI api.SMSAPI,
	fromSMSNumber string,
	branchClient branch.Client,
	rateLimiters ratelimit.KeyedRateLimiters,
	signer *sig.Signer,
	password string,
	analyticsLogger analytics.Logger,
	templateLoader *www.TemplateLoader,
	experimentIDs map[string]string,
	metricsRegistry metrics.Registry,
) {
	templateLoader.MustLoadTemplate("home/base.html", "base.html", nil)
	templateLoader.MustLoadTemplate("promotions/base.html", "home/base.html", nil)

	var protect func(http.Handler) http.Handler
	if password != "" {
		protect = PasswordProtectFilter(password, templateLoader)
	} else {
		protect = func(h http.Handler) http.Handler { return h }
	}

	r.Handle("/", protect(newStaticHandler(r, templateLoader, "home/home.html", "Spruce", nil)))
	r.Handle("/about", protect(newStaticHandler(r, templateLoader, "home/about.html", "About | Spruce", nil)))
	r.Handle("/conditions-treated", protect(newStaticHandler(r, templateLoader, "home/conditions.html", "Conditions Treated | Spruce", nil)))
	r.Handle("/contact", protect(newStaticHandler(r, templateLoader, "home/contact.html", "Contact | Spruce", nil)))
	r.Handle("/faq", protect(newStaticHandler(r, templateLoader, "home/faq.html", "FAQ | Spruce", faqCtx)))
	r.Handle("/free-visit-terms", protect(newStaticHandler(r, templateLoader, "home/free-visit-terms.html", "Free Visit Terms & Conditions | Spruce", nil)))
	r.Handle("/meet-the-doctors", protect(newStaticHandler(r, templateLoader, "home/meet-the-doctors.html", "Meet the Doctors | Spruce", nil)))
	r.Handle("/providers", protect(newStaticHandler(r, templateLoader, "home/providers.html", "For Providers | Spruce", nil)))
	r.Handle("/terms", protect(newStaticHandler(r, templateLoader, "home/terms.html", "Terms & Conditions | Spruce", nil)))

	// Email
	r.Handle("/e/optout", protect(newEmailOptoutHandler(dataAPI, authAPI, signer, templateLoader)))

	// Referrals
	r.Handle("/r/{code}", protect(newPromoClaimHandler(dataAPI, authAPI, analyticsLogger, templateLoader, experimentIDs["promo"])))
	r.Handle("/r/{code}/notify/state", protect(newPromoNotifyStateHandler(dataAPI, analyticsLogger, templateLoader, experimentIDs["promo"])))
	r.Handle("/r/{code}/notify/android", protect(newPromoNotifyAndroidHandler(dataAPI, analyticsLogger, templateLoader, experimentIDs["promo"])))

	// API
	r.Handle("/api/forms/{form:[0-9a-z-]+}", protect(NewFormsAPIHandler(dataAPI)))
	r.Handle("/api/textdownloadlink", protect(NewTextDownloadLinkAPIHandler(smsAPI, fromSMSNumber, branchClient, rateLimiters.Get("textdownloadlink"))))

	// Analytics
	ah := newAnalyticsHandler(analyticsLogger, metricsRegistry.Scope("analytics"))
	r.Handle("/a/events", ah)   // For javascript originating events
	r.Handle("/a/logo.png", ah) // For remote event tracking "pixels" (e.g. email)
}

func PasswordProtectFilter(pass string, templateLoader *www.TemplateLoader) func(http.Handler) http.Handler {
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
		} else {
			errorMsg = "Invalid password."
		}
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
	Sections []faqSection
}

type faqSection struct {
	Title     string
	Questions []faqQuestion
}

type faqQuestion struct {
	Question string
	Answer   template.HTML
}

var faqCtx = faqContext{
	Sections: []faqSection{
		{
			Title: "General",
			Questions: []faqQuestion{
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
						Getting treated by a doctor on Spruce is easy and fast. Once you’ve downloaded the app and created an account, all you have to do is take some photos of your symptoms and answer questions about your skin and medical history.
						</p>
						<p>
						A board-certified dermatologist will then carefully review your information and create a personalized treatment plan within 24 hours. Any prescriptions in your treatment plan will be sent digitally to your preferred pharmacy.
						</p>`,
				},
				{
					Question: "Where is Spruce available?",
					Answer:   "<p>Patients living in California, Florida, New York, and Pennsylvania can access Spruce. If you’re located somewhere else, don’t worry &mdash; we’ll be there soon.</p>",
				},
				{
					Question: "Can people under 18 use Spruce?",
					Answer:   "<p>At this time, Spruce is only accepting patients 18 and older.</p>",
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
			Questions: []faqQuestion{
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
			Questions: []faqQuestion{
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
			Questions: []faqQuestion{
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
			Questions: []faqQuestion{
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
			Questions: []faqQuestion{
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
	},
}
