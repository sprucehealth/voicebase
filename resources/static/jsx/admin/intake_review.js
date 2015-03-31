 var AdminAPI = require("./api.js");

module.exports = {
	/*
	Template expansion methods
	*/
	expandTemplate: function(template, status_cb) {
		// get rid of the fields of the template that a user entering the pathway framework
		// doesn't have to concern themselves with. Note that these fields will be repopulated
		// at the time of template submission
		delete(template.cost_item_type)
		delete(template.health_condition)
		delete(template.transitions)
		delete(template.is_templated)
		delete(template.visit_overview_header)
		delete(template.additional_message)
		delete(template.checkout)
		delete(template.submission_confirmation)

		// Reset our tag generation info
		this.generatedTags = {}

		for(section in template.sections) {
			template.sections[section] = this.sanitizeSection(template.sections[section], status_cb)
		}
		return template
	},

	sanitizeSection: function(section, status_cb) {
		uneededAttributes = ["description"]
		for(a in uneededAttributes) {
			delete(section[uneededAttributes[a]])
		}
		for(sc in section.screens) {
			section.screens[section] = this.sanitizeScreen(section.screens[sc], status_cb)
		}
		return section
	},

	sanitizeScreen: function(sc, status_cb) {
		uneededAttributes = ["description", "header_subtitle_has_tokens", "header_title_has_tokens"]
		for(a in uneededAttributes) {
			delete(sc[uneededAttributes[a]])
		}
		for(q in sc.questions) {
			sc.questions[q] = this.sanitizeQuestion(sc.questions[q], status_cb)
		}
		return sc
	},

	sanitizeQuestion: function(ques, status_cb) {
		version = 1
		language_id = 1
		tag = ques.question
		if(ques.version) {
			version = ques.version
		}
		if(ques.language_id) {
			language_id = ques.language_id
		}
		if(status_cb) {
			status_cb("Expanding question tag - " + tag)
		}
		AdminAPI.question(tag, language_id, version, function(success, data, error) {
				if(!success){
					throw error
				}
				ques.details = this.sanitizeQuestionDetails(data.versioned_question, status_cb)
			}.bind(this), false)
		uneededAttributes = ["language_id", "version", "question"]
		for(a in uneededAttributes) {
			delete(ques[uneededAttributes[a]])
		}
		if(ques.subquestions_config) {
			ques.subquestions_config = this.sanitizeSubquestionsConfig(ques.subquestions_config)
		}
		return ques
	},

	sanitizeSubquestionsConfig: function(sqc, status_cb) {
		for(scqq in sqc.questions) {
			sqc.questions[scqq] = this.sanitizeQuestion(sqc.questions[scqq], status_cb)
		}
		for(scqs in sqc.screens) {
			sqc.screens[scqs] = this.sanitizeScreen(sqc.screens[scqs], status_cb)
		}
		return sqc
	},

	sanitizeQuestionDetails: function(qd, status_cb) {
		uneededAttributes = ["id", "language_id", "status", "version"]
		for(a in uneededAttributes) {
			delete(qd[uneededAttributes[a]])
		}
		if(qd.required === true) {
			delete(qd.required)
		}

		var answer_groups = []
		if(Object.keys(qd.versioned_additional_question_fields).length === 0 || typeof qd.versioned_additional_question_fields == "undefined") {
			delete(qd.versioned_additional_question_fields)
		} else {
			// keep track of whether or not answer groups are defined for the question
			if (typeof qd.versioned_additional_question_fields.answer_groups != "undefined") {
				answer_groups = qd.versioned_additional_question_fields.answer_groups
			}

			qd.additional_question_fields = qd.versioned_additional_question_fields
			delete(qd.versioned_additional_question_fields)
		}

		if(qd.versioned_answers.length != 0) {
			// if answer groups are defined in the additional fields
			// then make it a first class citizen in the details object
			// as it was entered
			if (answer_groups.length > 0) {

				var j = 0
				qd.answer_groups = []
				for (agi in answer_groups) {

					answer_group = {
						title: answer_groups[agi].title,
						answers: []
					}

					// go through the defined count to append the appropriate number of answers
					// to the group
					for (i = 0; i < answer_groups[agi].count; i++) {
						answer_group.answers.push(this.sanitizeAnswer(qd.versioned_answers[j], qd, status_cb))
						j++
					}

					qd.answer_groups.push(answer_group)
				}
				// remove the answer group from the additional fields as it will be
				// converted back into this format at the time of submission
				delete(qd.additional_question_fields.answer_groups)
			} else {
				qd.answers = []
				for(va in qd.versioned_answers) {
					qd.answers.push(this.sanitizeAnswer(qd.versioned_answers[va], qd, status_cb))
				}
			}
		}
		if(qd.versioned_photo_slots.length != 0) {
			qd.photo_slots = []
			for(vps in qd.versioned_photo_slots) {
				qd.photo_slots.push(this.sanitizePhotoSlot(qd.versioned_photo_slots[vps], status_cb))
			}
		}
		delete(qd.versioned_photo_slots)
		delete(qd.versioned_answers)
		return qd
	},

	sanitizeAnswer: function(ans, qd, status_cb) {
		uneededAttributes = ["id", "language_id", "ordering", "question_id", "status"]
		for(a in uneededAttributes) {
			delete(ans[uneededAttributes[a]])
		}
		if(ans.type == this.defaultAnswerTypeforQuestion(qd.type, status_cb)){
			delete(ans.type)
		}
		if(typeof ans.client_data != "undefined" && Object.keys(ans.client_data || {}).length === 0) {
			delete(ans.client_data)
		}
		return ans
	},

	sanitizePhotoSlot: function(ps, status_cb) {
		var uneededAttributes = ["id", "language_id", "ordering", "question_id", "status"]
		for(a in uneededAttributes) {
			delete(ps[uneededAttributes[a]])
		}
		if(ps.type == "photo_slot_standard" && ps.required){
			delete(ps.required)
		}
		if(!ps.client_data || ps.client_data == null || Object.keys(ps.client_data).length == 0){
			delete(ps.client_data)
		}
		return ps
	},

	/*
	Review Generation Methds
	*/

	generateReview: function(intake, pathway) {
		var review = {}
		//Reset our tag info
		this.generatedTags = {}
		review.visit_review = {type: "d_visit_review:sections_list", sections: []}
		review.visit_review.sections.push(this.alertSection())
		for(sec in intake.sections) {
			for(screen_view in intake.sections[sec].screens) {
				if(intake.sections[sec].screens[screen_view].type == "screen_type_photo" || this.containsPhotoQuestions(intake.sections[sec].screens[screen_view])){
					review.visit_review.sections.push(this.parsePhotoScreen(intake.sections[sec].screens[screen_view], pathway))
					delete(intake.sections[sec].screens[screen_view])
				}
			}
		}
		for(sec in intake.sections) {
			section = this.parseSection(intake.sections[sec], pathway)
			if(section.subsections.length > 0){
				review.visit_review.sections.push(section)
			}
		}
		review.visit_review.sections.push(this.visitMessageSection())
		return review
	},

	parseSection: function(section, pathway) {
		this.required(section, ["section_title", "transition_to_message"], "Section")
		var review_section = {title: section.section_title, type: "d_visit_review:standard_section", subsections: []}
		if(typeof section.subsections == "undefined") {
			this.required(section, ["screens"], "Section without Subsections")
			var subsection = this.generateReviewSubsectionFromScreens(section, pathway, section.section_title + " Questions")
			if(subsection.rows.length != 0){
				review_section.subsections.push(subsection)
			}
		} else {
			this.required(section, ["subsections"], "Section without Screens")
			for(ss in section.subsections) {
				this.required(section.subsections[ss], ["title", "screens"], "Subsection")
				var subsection = this.generateReviewSubsectionFromScreens(section.subsections[ss], pathway, section.subsections[ss].title)
				if(subsection.rows.length != 0){
					review_section.subsections.push(subsection)
				}
			}
		}
		return review_section
	},

	generateReviewSubsectionFromScreens: function(obj, pathway, title) {
		var question_subsection = {
			rows: [],
			title: title,
			type: "d_visit_review:standard_subsection"
		}

		var contentKeys = []
		for(scr in obj.screens) {
			question_subsection.rows = question_subsection.rows.concat(this.parseQuestionScreen(obj.screens[scr], pathway))

			// idenfify all the content keys within the subsection
			for (q in obj.screens[scr].questions) {
				contentKeys.push(obj.screens[scr].questions[q].details.tag+":answers")
			}
		}

		// only show the subsection to the doctor if atleast one of the questions has been answered
		question_subsection.content_config = {
			condition: {
				op: "any_key_exists",
				keys: contentKeys
			}
		}
		return question_subsection
	},

	parsePhotoScreen: function(screen_view, pathway) {
		section = {
			title: screen_view.header_summary,
			type: "d_visit_review:standard_photo_section",
			subsections: []
		}

		var contentKeys = []
		for(question in screen_view.questions){
			if(!screen_view.questions[question].details.tag) {
				screen_view.questions[question].details.tag = this.tagFromText(screen_view.questions[question].details.text)
			}
			tag = this.transformQuestionTag(screen_view.questions[question].details.tag, pathway, screen_view.questions[question].details.global)
			contentKeys.push(tag+":photos")
			section.subsections.push(this.photoSubSection(tag))
		}
		section.content_config = {
			condition : {
				op: "any_key_exists",
				keys: contentKeys
			}
		}

		return section
	},

	photoSubSection: function(tag) {
		return {
			content_config: {
				condition: {
						op: "key_exists",
						key: tag+":photos"
				}
			},
			type: "d_visit_review:standard_photo_subsection",
			view: {
				type: "d_visit_review:title_photos_items_list",
				content_config: {
					key: tag+":photos"
				}
			}
		}
	},

	parseQuestionScreen: function(screen_view, pathway) {
		var question_rows = []
		for(question in screen_view.questions){
			question_rows.push(this.parseQuestion(screen_view.questions[question].details, screen_view.questions[question].subquestions_config, pathway))
		}
		return question_rows
	},

	parseQuestion: function(ques, sc, pathway) {
		if(!ques.tag) {
			ques.tag = this.tagFromText(ques.text, pathway)
		}

		ques.tag = this.transformQuestionTag(ques.tag, pathway, ques.global)

		var row = {}
		if(sc) {
			row = this.parseMultiPart(ques)
		} else {
			if(ques.type === "q_type_single_select" || ques.type === "q_type_segmented_control") {
				row = this.parseSingleSelect(ques)
			} else if (ques.type === "q_type_multiple_choice") {
				row = this.parseMultipleChoice(ques)
			} else if (ques.type === "q_type_autocomplete") {
				row = this.parseAutoComplete(ques)
			} else if (ques.type === "q_type_free_text") {
				row = this.parseFreeText(ques)
			}
		}
		return row
	},

	parseFreeText: function(ques) {
		return {
			content_config : {
				condition: {
					op: "key_exists",
					key: ques.tag+":answers"
				}
			},
			 left_view: {
					content_config: {
							key: ques.tag+":question_summary"
					},
					type: "d_visit_review:title_labels_list"
			},
			right_view: {
					content_config: {
							key: ques.tag+":answers"
					},
					type: "d_visit_review:content_labels_list",
			},
			type: "d_visit_review:standard_two_column_row"
		}
	},

	parseMultiPart: function(ques) {
		return {
			content_config : {
				condition: {
					op: "key_exists",
					key: ques.tag+":answers"
				}
			},
			left_view: {
				content_config: {
					key: ques.tag+":question_summary"
				},
				type: "d_visit_review:title_labels_list"
			},
			right_view: {
				content_config: {
					key: ques.tag+":answers"
				},
				empty_state_view: {
					content_config: {
						key: ques.tag+":empty_state_text"
					},
					type: "d_visit_review:empty_label"
				},
				type: "d_visit_review:title_subitems_description_content_labels_divided_items_list"
			},
			type: "d_visit_review:standard_two_column_row"
		}
	},

	parseAutoComplete: function(ques) {
		return this.parseSingleSelect(ques)
	},

	visitMessageSection: function(ques) {
		return {
			subsections: [
				{
					condition: {
						key: "visit_message",
						op: "key_exists"
					},
					rows: [
						{
							type: "d_visit_review:standard_one_column_row",
							view: {
								content_config: {
									key: "visit_message"
								},
								empty_state_view: {
									content_config: {
										key: "visit_message:empty_state_text"
									},
									type: "d_visit_review:empty_label"
								},
								type: "d_visit_review:content_labels_list"
							}
						}
					],
					title: "Additional Information from Patient",
					type: "d_visit_review:standard_subsection"
				}
			],
			title: "Visit Message",
			type: "d_visit_review:standard_section"
		}
	},

	parseSingleSelect: function(ques) {
		return {
			content_config : {
				condition: {
					op: "key_exists",
					key: ques.tag+":answers"
				}
			},
			left_view: {
					content_config: {
							key: ques.tag+":question_summary"
					},
					type: "d_visit_review:title_labels_list"
			},
			right_view: {
					content_config: {
							key: ques.tag+":answers"
					},
					type: "d_visit_review:content_labels_list"
			},
			type: "d_visit_review:standard_two_column_row"
		}
	},

	parseMultipleChoice: function(ques) {
		return {
		 content_config : {
				condition: {
					op: "key_exists",
					key: ques.tag+":answers"
				}
			},
			 left_view: {
					content_config: {
							key: ques.tag+":question_summary"
					},
					type: "d_visit_review:title_labels_list"
			},
			right_view: {
					content_config: {
							key: ques.tag+":answers"
					},
					type: "d_visit_review:check_x_items_list"
			},
			type: "d_visit_review:standard_two_column_row"
		}
	},

	alertSection: function() {
		return {
			subsections: [
				{
					rows: [
						{
							condition: {
								op: "key_exists",
								key: "patient_visit_alerts"
							},
							type: "d_visit_review:standard_one_column_row",
							view: {
								content_config: {
									"key": "patient_visit_alerts"
								},
								empty_state_view: {
									content_config: {
										key: "patient_visit_alerts:empty_state_text"
									},
									type: "d_visit_review:empty_label"
								},
								type: "d_visit_review:alert_labels_list"
							}
						}
					],
					title: "Alerts",
					type: "d_visit_review:standard_subsection"
				}
			],
			title: "Alerts",
			type: "d_visit_review:standard_section"
		}
	},

	submitLayout: function(intake, review, pathway, status_cb) {
		//Reset our tag info
		this.generatedTags = {}
		try {
			intake = this.transformIntake(intake, pathway, status_cb)
		} catch (e) {
			e.message = "Intake Transformation Error: " + e.message
			throw e
		}
		review.health_condition = intake.health_condition
		review.cost_item_type = intake.cost_item_type
		console.log("Transformed Intake ", intake, "Review", review)
		fd = new FormData()
		fd.append("intake", new Blob([JSON.stringify(intake)], { type: "application/json" }))
		fd.append("review", new Blob([JSON.stringify(review)], { type: "application/json" }))
		// TODO:REMOVE - HACK! Remove this one we get better versioning support
		fd.append("doctor_app_version", "1.2.0")
		fd.append("patient_app_version", "1.2.0")
		fd.append("platform", "iOS")
		if(status_cb){
			status_cb("Uploading transformed layout")
		}
		AdminAPI.layoutUpload(fd, function(success, data, error) {
			if(!success){
				error.message = "Intake Submission Error: " + error.message
				throw error
			}
		}, false)
	},

	/*
	{
		{
			"sections": [
				{
					“auto|section”: “An identifier for the section - If not provided one will be generated”
					“auto|section_id”:   “The new identifier for the section - If not provided the `section` attribute will be use”
					“section_title”: “The section title to be presented to the client”
					“transition_to_message”: “The message to display to the user when transitioning into this section”
					"screens": [
						{
							"auto|header_title_has_tokens": false, // true|false - representing if this string used tokens
							"auto|header_subtitle_has_tokens": false, // true|false - representing if this string used tokens
							"auto|header_summary": "The summary to present to the user in relation to photo slots"
							"optional|header_subtitle": "The subtitle of the screen",
							"optional|header_title": "The title of the screen",

							"questions": [
									{
										"optional|condition" : {
											"op": "answer_equals_exact | answer_contains_any | answer_contains_all | gender_equals | and | or",
											"*gender" : "male|female", // Required if gender_equals is the op
											"*operands" : [{ // Required if selected operation is and | or
													"op" : "answer_equals_exact | answer_contains_any | answer_contains_all | gender_equals | and | or",
													// this is a recursive definition of a condition object
											}],
											"*auto|question_tag": "The tag of the question that you are referencing in this conditional", // Required if the selected 'op' is answer_xxxxx
											"*answer_tags": ["List of the answer tags to evaluate in this conditional"] // Required if the selected 'op' is answer_xxxxx
										},

										"details": {
											"auto|required": true, // true|false - representing if this question is required to be answered by the user
											"auto|unique|tag": "Generated if not specified. Should be specified if referenced elsewhere. Will have global|pathway_tag prepended",
											"auto|text_has_tokens": false, // true|false - representing if this string used tokens,
											"auto|summary_text": "Generated is not specified using the question text"
											"optional|global": false, // true|false - representing if this question should be scoped to the pathway or globally. A question is scoped globally if it belongs to the patient’s medical history.,
											"optional|to_prefill": false, // true|false - representing if this question should have its answer prepopulated from historical data
											"optional|to_alert": false, // true|false - representing if this question should be flagged to the reviewer (highlighted)
											"optional|alert_text": "The highlighted text to display to the reviewer if 'to_alert' is true",

											"text": "The literal question text shown to the user",
											"type": "q_type_multiple_choice|q_type_free_text|q_type_single_select|q_type_segmented_control|q_type_autocomplete|q_type_photo_section",
											"auto|answers": [
												{
													"auto|summary_text": "The text shows in review contexts - will be auto generated from the literal text",
													"auto|tag": "Generated if not specified. Should be specified if referenced elsewhere. Will have global|pathway_tag prepended.",
													"auto|type": "a_type_multiple_choice|a_type_segmented_control|a_type_multiple_choice_none|a_type_multiple_choice_other_free_text",
													"optional|to_alert": false, // true|false - representing if this answer should be flagged to the reviewer (highlighted),
													"optional|client_data": "Data pertaining to the answer for the client to consume (Eg. help popovers)"
													"text": "The literal answer text shown to the user",
												},
												{
													// Other answers answers
												}
											],
											"auto|photo_slots": [
												{
													"optional|type": "The type of photo slot to be presented to the user",
													"optional|client_data": "Data describing the photo slot to be utilized by the client"
													"name": "The name to associate with this photo slot"
												}
											],
											"auto|additional_question_fields": {
												"optional|empty_state_text": "Text to populate the review with when an optional question is left empty",
												"optional|placeholder_text": "Text to populate before any contents have been added by the user. Shown in gray and should generally be used with free text or single entry questions",
												"optional|other_answer_placeholder_text": "Placeholder text to populate the 'other' section of a multi select question", // Example. 'Type another treatment name'
												"optional|add_text": "*Used with autocomplete questions - Don't look here. These aren't the droids you're looking for.",
												"optional|add_button_text": "*Used with autocomplete questions - Don't look here. These aren't the droids you're looking for.",
												"optional|save_button_text": "*Used with autocomplete questions - Don't look here. These aren't the droids you're looking for.",
												"optional|remove_button_text": "*Used with autocomplete questions - Don't look here. These aren't the droids you're looking for.",
												"optional|allows_multiple_sections": false, // true|false - Used with a photo section question to allow that section to be added multiple times
												"optional|user_defined_section_title": false // true|false - User provided title for the section.
											}
										},
										"optional|subquestions_config": {
											"optional|screens": [], // Must contain a screen object, parent question must be a q_type_multiple_choice question. Generally used with header title tokens or question title tokens that allow the parent answer text to be inserted into the header title or question title.
											"optional|questions": [] // Parent question must be a q_type_autocomplete - Don't use otherwise. Contains an array of question objects. The parent question_id of these questions will be auto completed.
										}
									}
							],
							"screen_type": "screen_type_photo",
						},
						{
							"screen_type": "screen_type_pharmacy"
						}
					],
				}
			]
		}
	}
	*/

	// HACK! Account for the acne special case
	specialCaseSku: function(pathway) {
		if(pathway == "health_condition_acne") {
			return "acne"
		}
		return pathway
	},

	required: function(obj, fields, type_desc) {
		for(field in fields) {
			if(typeof obj[fields[field]] == "undefined") {
				var failed_obj = jsyaml.safeDump(obj)
				var json_info = (failed_obj.substring(0, failed_obj.length > 480 ? 480 : failed_obj.length)) + (failed_obj.length > 480 ? "\n[truncated]" : "")
				throw <span>Field `{fields[field]}` required but missing for object `{type_desc}` <br/>----- Object Contents -----<br/> <pre> {json_info} </pre></span>
			}
		}
	},

	randomString: function(length) {
		text = "";
		possible = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789";

		for(r=0; r < length; r++) {
			text += possible.charAt(Math.floor(Math.random() * possible.length));
		}

		return text;
	},

	transformIntake: function(intake, pathway, status_cb) {
		this.required(intake, ["sections"], "Intake")

		// keep track of global questions
		var globalQuestions = {}
		for(i in intake.sections) {
			intake.sections[i] = this.transformSection(intake.sections[i], globalQuestions, pathway, status_cb)
			if(i == (intake.sections.length - 1)) {
				if(intake.sections[i].screens[intake.sections[i].screens.length-1].screen_type != "screen_type_pharmacy") {
					intake.sections[i].screens.push(this.pharmacyScreen())
				}
			}
		}
		intake = this.populateIntakeMetadata(intake, pathway)
		return intake
	},

	populateIntakeMetadata: function(intake, pathway) {
		intake.health_condition = pathway
		intake.cost_item_type = this.specialCaseSku(pathway) + "_visit"

		if(!intake.transitions) {
			intake.transitions = this.generateTransitions(intake.sections)
		}

		intake.is_templated = true
		intake.visit_overview_header = {
			title: "{{.CaseName}} Visit",
			subtitle: "With {{.Doctor.Description}}",
			icon_url: "{{.Doctor.SmallThumbnailURL}}"
		}
		intake.additional_message = {
			title: "Is there anything else you’d like to ask or share with {{.Doctor.ShortDisplayName}}?",
			placeholder: "It’s optional but this is your chance to let the doctor know what’s on your mind."
		}
		intake.checkout = {
			header_image_url: "{{.Doctor.SmallThumbnailURL}}",
			header_text: "{{.CheckoutHeaderText}}",
			footer_text: "There are no surprise medical bills with Spruce. If you're unsatisfied with your visit, we'll refund the full cost."
		}
		intake.submission_confirmation = {
			title: "Visit Submitted!",
			top_text: "Your {{.CaseName | toLower}} visit has been submitted.",
			bottom_text: "{{.SubmissionConfirmationText}}",
			button_title: "Continue"
		}

		return intake
	},

	generateTransitions: function(sections) {
		transitions = []
		first = true
		for(section in sections) {
			transitions.push(this.newTransition(sections[section].transition_to_message, first ? "Begin" : "Continue"))
			first = false
		}
		transitions.push(this.newTransition("That's all the information your doctor will need!", "Continue"))
		return transitions
	},

	newTransition: function(message, button_text) {
		return {
			"buttons": [
				{
					"button_text": button_text,
					"style": "filled",
					"tap_url": "spruce:///action/view_next_visit_section"
				}
			],
			"message": message
		}
	},

	transformSection: function(section, globalQuestions, pathway, status_cb) {
		this.required(section, ["section_title", "transition_to_message"], "Section")
		if(typeof section.subsections == "undefined"){
			this.required(section, ["screens"], "Section without subsections")
		} else {
			this.required(section, ["subsections"], "Section without screens")
		}
		if(typeof section.screens != "undefined" && typeof section.subsections != "undefined") {
			throw new Error("A section cannot contain both subsections and screens.")
		}
		if(typeof section.subsections == "undefined"){
			for(sc in section.screens) {
				section.screens[sc] = this.transformScreen(section.screens[sc], globalQuestions, pathway, status_cb)
			}
		} else {
			section.screens = []
			for(s in section.subsections) {
				for(sc in section.subsections[s].screens) {
					section.screens.push(this.transformScreen(section.subsections[s].screens[sc], globalQuestions, pathway, status_cb))
				}
			}
			delete(section.subsections)
		}
		if(typeof section.section == "undefined") {
			section.section = this.randomString(12)
		}
		if(typeof section.section_id == "undefined") {
			section.section_id = section.section
		}
		return section
	},

	transformScreen: function(sc, globalQuestions, pathway, status_cb) {
		if(!sc.questions) {
			this.required(sc, ["screen_type"], "Screen without Questions")
		} else if (this.containsPhotoQuestions(sc)) {
			this.required(sc, ["header_title", "header_summary"], "Screen with Photo Questions")
			if(!sc.screen_type){
				sc.screen_type = "screen_type_photo"
			} else if (sc.screen_type != "screen_type_photo") {
				throw new Error("Sections containing photo questions must have type screen_type_photo. Found " + sc.type)
			}
		}
		if(sc.header_title) {
			sc.header_title_has_tokens = this.token_pattern.test(sc.header_title)
		}
		if(sc.header_subtitle) {
			sc.header_subtitle_has_tokens = this.token_pattern.test(sc.header_subtitle)
		}
		for(si in sc.questions) {
			sc.questions[si] = this.transformQuestion(sc.questions[si], globalQuestions, pathway, status_cb)
		}

		if (sc.condition) {
				sc.condition = this.transformCondition(sc.condition, globalQuestions, pathway)
		}

		switch (sc.screen_type) {
			case "screen_type_warning_popup":
				this.validateWarningPopupScreen(sc)
				break
			case "screen_type_triage":
				this.validateTriageScreen(sc)
				break
		}

		return sc
	},

	containsPhotoQuestions: function(scr) {
		if(typeof scr.questions == "undefined") return false
		for(q in scr.questions) {
			this.required(scr.questions[q], ["details"], "Question")
			if(scr.questions[q].details.type == "q_type_photo_section") return true
		}
		return false
	},

	validateTriageScreen: function(sc) {
		this.required(sc, ["screen_type", "body", "condition", "content_header_title", "screen_title", "bottom_button_title"], "Triage screen")
		this.validateBody(sc.body)
		this.validateCondition(sc.condition)

		// this screen type cannot have any questions defined
		if (sc.questions) {
			throw new Error("Screen defined as type screen_type_triage cannot have any questions")
		}
	},

	validateWarningPopupScreen: function(sc) {
		this.required(sc, ["screen_type", "body", "condition", "content_header_title"], "Warning popup screen")
		this.validateBody(sc.body)
		this.validateCondition(sc.condition)

		// this screen type cannot have any questions defined
		if (sc.questions) {
			 throw new Error("Screen defined as type screen_type_warning_popup should have no questions")
		}
	},

	validateBody: function(body) {
		this.required(body, ["text"], "Body definition")
		if (body.Button) {
			this.validateButton(body.Button)
		}
	},

	validateButton: function(button) {
		this.required(button, ["button_text", "tap_url", "style"], "Button definition")
	},

	validateCondition: function(condition) {
		this.required(condition, ["op"])
		switch (condition.op) {
			case "answer_equals":
			case "answer_equals_exact":
			case "answer_contains_any":
			case "answer_contains_all":
					this.required(condition, ["question", "potential_answers"], "Question/Answer conditional")
					break

			case "gender_equals":
					this.required(condition, ["gender"], "Gender conditional")
					break

			case "and":
			case "or":
			case "not":
					this.required(condition, ["operands"], "Logical conditional")
					// validate operands (which are conditionals themselves)
					for (var i in condition.operands) {
						this.validateCondition(condition.operands[i])
					}
					break

			default:
				throw new Error("Unsupported condition type: " + condition.op)
		}
	},

	// a value is considered scoped if its either marked as being global
	// or is of the form <prefix><pathway>_<tag>
	isScoped: function(value, pathway, prefix, global) {
		if (global) {
				return value
		}
		pathway_regex = new RegExp(prefix + pathway + "_")
		return pathway_regex.test(value)
	},

	transformAnswerTag: function(tag, pathway, global) {
		if(!this.isScoped(tag, pathway, "a_", global)){
			return this.scopeTag(tag, pathway, global, "a_")
		}
		return tag
	},

	transformQuestionTag: function(tag, pathway, global) {
		if(!this.isScoped(tag, pathway, "q_", global)){
			return this.scopeTag(tag, pathway, global, "q_")
		}
		return tag
	},

	scopeTag: function(tag, pathway, global, prefix) {
		return global ? tag : prefix + pathway + "_" + tag
	},

	generatedTags: {},

	tagFromText: function(text, pathway) {
		tag = text.toLowerCase().replace(/ /g,"_").replace(/[\.,-\/#!$%\^&\*;:{}=\-`~()<>\?]/g,"")
		v = this.generatedTags[tag]
		if(typeof v == "undefined"){
			v = 1
		} else {
			v = v + 1
			tag = tag + "_" + v
		}
		this.generatedTags[tag] = v
		return tag
	},

	transformQuestion: function(ques, globalQuestions, pathway, status_cb) {
		this.required(ques, ["details"], "Question")
		this.required(ques.details, ["text","type"], "Question.Details")
		if(typeof ques.details.required == "undefined") {
			ques.details.required = true
		}

		if(ques.condition) {
			ques.condition = this.transformCondition(ques.condition, globalQuestions, pathway)
		}

		if(!ques.details.tag) {
			ques.details.tag = this.tagFromText(ques.details.text, pathway)
		}

		if(!ques.details.summary_text){
			ques.details.summary_text = ques.details.text
		}

		ques.details.tag = this.transformQuestionTag(ques.details.tag, pathway, ques.details.global)
		if (ques.details.global) {
			globalQuestions[ques.details.tag] = true	
		}
		
		if(typeof ques.details.text_has_tokens == "undefined") {
			ques.details.text_has_tokens = this.token_pattern.test(ques.details.text)
		}
		if(!ques.details.additional_question_fields) {
			ques.details.additional_question_fields = {}
		}
		if(ques.details.type == "q_type_photo_section") {
			this.required(["photo_slots"])
			if(this.answers) {
				throw new Error("Questions of type q_type_photo_section may not contain an 'answers' section. Only 'photo_slots'")
			}
		}
		if(ques.subquestions_config) {
			ques.subquestions_config = this.transformSubquestionsConfig(ques.subquestions_config, globalQuestions, pathway, status_cb)
		}

		if (ques.condition) {
			this.validateCondition(ques.condition)
		}

		// transform any answer groups into an array of answers
		// with the appropriate information in the additional fields
		if (typeof ques.details.answer_groups != "undefined") {
			var answer_groups = []
			ques.details.answers = []
			for(agi in ques.details.answer_groups) {
				answer_group = {
					title: ques.details.answer_groups[agi].title,
					count: 0
				}
				for (ai in ques.details.answer_groups[agi].answers) {
					answer_group.count++
					ques.details.answers.push(ques.details.answer_groups[agi].answers[ai])
				}
				answer_groups.push(answer_group)
			}

			if (typeof ques.details.additional_question_fields == "undefined") {
				ques.details.additional_question_fields = {}
			}

			ques.details.additional_question_fields.answer_groups = answer_groups
			delete(ques.details.answer_groups)
		}

		ques.details.versioned_additional_question_fields = ques.details.additional_question_fields
		delete(ques.details.additional_question_fields)

		//TODO: In the future we'll need to have the tool allow the user to choose languages
		ques.details.language_id = "1"
		for(a in ques.details.answers) {
			ques.details.answers[a] = this.transformAnswer(ques.details.answers[a], pathway, ques, a)
		}
		ques.details.versioned_answers = ques.details.answers ? ques.details.answers : []
		delete(ques.details.answers)

		for(ps in ques.details.photo_slots) {
			ques.details.photo_slots[ps] = this.transformPhotoSlot(ques.details.photo_slots[ps], pathway)
		}
		ques.details.versioned_photo_slots = ques.details.photo_slots ? ques.details.photo_slots : []
		delete(ques.details.photo_slots)

		if(status_cb){
			status_cb("Versioning question - ", ques.details.tag)
		}
		tag_version = this.submitQuestion(ques.details)

		ques.to_prefill = ques.details.to_prefill
		delete(ques.details)
		ques.question = tag_version.tag
		ques.version = tag_version.version
		return ques
	},

	defaultAnswerTypeforQuestion: function(question_type) {
		switch(question_type) {
				case "q_type_multiple_choice":
					return "a_type_multiple_choice"
				case "q_type_single_select":
					return "a_type_multiple_choice"
				case "q_type_segmented_control":
					return "a_type_segmented_control"
				case "q_type_segmented_control":
					return "a_type_segmented_control"
				default:
					return null
			}
	},

	transformAnswer: function(ans, pathway, ques, order) {
		this.required(ans, ["text"], "Answer")
		if(!ans.summary_text) {
			ans.summary_text = ans.text
		}

		if(!ans.tag) {
			ans.tag = this.tagFromText(ans.text, pathway)
		}

		ans.tag = this.transformAnswerTag(ans.tag, pathway, ques.details.global)

		if(!ans.type) {
			ans.type = this.defaultAnswerTypeforQuestion(ques.details.type)
			if(ans.type == null) throw new Error("Unknown question type " + ques.details.type + " for untyped answer")
		}
		//TODO: In the future we'll need to have the tool allow the user to choose languages
		ans.language_id = "1"
		ans.status = "ACTIVE"
		ans.ordering = order
		return ans
	},

	transformCondition: function(condition, globalQuestions, pathway) {
		var isGlobal = (condition.question in globalQuestions)

		if(condition.question && !this.isScoped(condition.question, pathway, "q_", isGlobal)) {
			condition.question = this.transformQuestionTag(condition.question, pathway, isGlobal)
		}
		for(pa in condition.potential_answers) {
			if(!this.isScoped(condition.potential_answers[pa], pathway, "a_", isGlobal)) {
				condition.potential_answers[pa] = this.transformAnswerTag(condition.potential_answers[pa], pathway, isGlobal)
			}
		}

		// iterate through each of the operands and recursively transform the condition
		// if it exists
		if (typeof condition.operands != "undefined") {
			for (oi in condition.operands) {
				condition.operands[oi] = this.transformCondition(condition.operands[oi], globalQuestions, pathway)
			}
		}
		return condition
	},

	transformPhotoSlot: function(ps, pathway) {
		this.required(ps, ["name"], "Photo Slot")
		if(!ps.client_data) {
			ps.client_data = {}
		}
		if(typeof ps.required == "undefined") {
			ps.required = true
		}
		return ps
	},

	transformSubquestionsConfig: function(sqc, globalQuestions, pathway, status_cb) {
		for(sqcs in sqc.screens) {
			sqc.screens[sqcs] = this.transformScreen(sqc.screens[sqcs], globalQuestions, pathway, status_cb)
		}
		for(sqcq in sqc.questions) {
			sqc.questions[sqcq] = this.transformQuestion(sqc.questions[sqcq], globalQuestions, pathway, status_cb)
		}
		return sqc
	},

	pharmacyScreen: function() {
		return {
			screen_type: "screen_type_pharmacy"
		}
	},

	submitQuestion: function(ques) {
		tag_version = {}
		AdminAPI.submitQuestion(ques, function(success, data, error) {
			if(!success){
				throw error
			}
			tag_version = {tag: data.versioned_question.tag, version: data.versioned_question.version}
		}, false)
		return tag_version
	},

	token_pattern:  /<\w+>/,
}
