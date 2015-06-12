/* @flow */

var objectAssign = require('object-assign');

module.exports = {
	StatusNotFound: 404,

	ajax: function(params: ajaxParams, cb: ajaxCB, async?: bool) {
		jQuery.ajax(objectAssign(params, {
			async: (async == true || async == null),
			url: "/admin/api" + params.url,
			success: function(data) {
				cb(true, data, noError, null);
			},
			error: function(jqXHR) {
				// Since success=false already is used to signal that data can be null
				// we can force flow to not throw errors on missing null checks on data.
				var x: any = null;
				cb(false, x, parseError(jqXHR), jqXHR);
			}
		}));
	},

	// Accounts

	account: function(id: string, cb: ajaxCB) {
		this.ajax({
			type: "GET",
			url: "/accounts/" + encodeURIComponent(id),
			dataType: "json"
		}, cb);
	},
	accountPhoneNumbers: function(id: string, cb: ajaxCB) {
		this.ajax({
			type: "GET",
			url: "/accounts/" + encodeURIComponent(id) + "/phones",
			dataType: "json"
		}, cb);
	},
	updateAccount: function(id: string, account: any, cb: ajaxCB) {
		this.ajax({
			type: "PATCH",
			contentType: "application/json",
			url: "/accounts/" + id,
			data: JSON.stringify(account),
			dataType: "json"
		}, cb);
	},

	// Doctors / care providers

	doctor: function(id: string, cb: ajaxCB) {
		this.ajax({
			type: "GET",
			url: "/doctors/" + encodeURIComponent(id),
			dataType: "json"
		}, cb);
	},
	doctorAttributes: function(id: string, cb: ajaxCB) {
		this.ajax({
			type: "GET",
			url: "/doctors/" + encodeURIComponent(id) + "/attributes",
			dataType: "json"
		}, cb);
	},
	doctorFavoriteTreatmentPlans: function(id: string, cb: ajaxCB) {
		this.ajax({
			type: "GET",
			url: "/doctors/" + encodeURIComponent(id) + "/treatment_plan/favorite",
			dataType: "json"
		}, cb);
	},
	medicalLicenses: function(doctorID: string, cb: ajaxCB) {
		this.ajax({
			type: "GET",
			url: "/doctors/" + doctorID + "/licenses",
			dataType: "json"
		}, cb);
	},
	updateMedicalLicenses: function(doctorID: string, licenses: any, cb: ajaxCB) {
		this.ajax({
			type: "PUT",
			contentType: "application/json",
			url: "/doctors/" + doctorID + "/licenses",
			data: JSON.stringify({"licenses": licenses}),
			dataType: "json"
		}, cb);
	},
	searchDoctors: function(query: string, cb: ajaxCB) {
		this.ajax({
			type: "GET",
			url: "/doctors?q=" + encodeURIComponent(query),
			dataType: "json"
		}, cb);
	},
	careProviderProfile: function(doctorID: string, cb: ajaxCB) {
		this.ajax({
			type: "GET",
			url: "/doctors/" + doctorID + "/profile",
			dataType: "json"
		}, cb);
	},
	updateCareProviderProfile: function(doctorID: string, profile: any, cb: ajaxCB) {
		this.ajax({
			type: "PUT",
			contentType: "application/json",
			url: "/doctors/" + doctorID + "/profile",
			data: JSON.stringify(profile),
			dataType: "json"
		}, cb);
	},
	doctorOnboarding: function(cb: ajaxCB) {
		this.ajax({
			type: "GET",
			url: "/dronboarding",
			dataType: "json"
		}, cb);
	},
	doctorProfileImageURL: function(id: string, pImageType: string): string {
		return "/admin/api/doctors/" + encodeURIComponent(id) + "/profile_image/" + encodeURIComponent(pImageType);
	},
	updateDoctorProfileImage: function(id: string, pImageType: string, formData: any, cb: ajaxCB) {
		this.ajax({
			type: "PUT",
			cache: false,
			contentType: false,
			processData: false,
			url: "/doctors/" + encodeURIComponent(id) + "/profile_image/" + encodeURIComponent(pImageType),
			data: formData
		}, cb);
	},
	careProviderStatePathwayMappings: function(params: {state: ?string; pathwayTag: ?string}, cb: ajaxCB) {
		var query = [];
		if (params.state) {
			query.push("state=" + encodeURIComponent(params.state));
		}
		if (params.pathwayTag) {
			query.push("pathway_tag=" + encodeURIComponent(params.pathwayTag));
		}
		this.ajax({
			type: "GET",
			url: "/care_providers/state_pathway_mappings?" + query.join("&"),
			dataType: "json"
		}, cb);
	},
	careProviderStatePathwayMappingsSummary: function(cb: ajaxCB) {
		this.ajax({
			type: "GET",
			url: "/care_providers/state_pathway_mappings/summary",
			dataType: "json"
		}, cb);
	},
	careProviderEligibility: function(providerID: string, cb: ajaxCB) {
		this.ajax({
			type: "GET",
			url: "/doctors/" + encodeURIComponent(providerID) + "/eligibility",
			dataType: "json"
		}, cb);
	},
	updateCareProviderEligiblity: function(providerID: string, update: any, cb: ajaxCB) {
		this.ajax({
			type: "PATCH",
			contentType: "application/json",
			url: "/doctors/" + encodeURIComponent(providerID) + "/eligibility",
			data: JSON.stringify(update),
			dataType: "json"
		}, cb);
	},

	// Guides

	resourceGuide: function(id: string, cb: ajaxCB) {
		this.ajax({
			type: "GET",
			url: "/guides/resources/" + id,
			dataType: "json"
		}, cb);
	},
	resourceGuidesList: function(withLayouts: bool, sectionsOnly: bool, cb: ajaxCB) {
		var params = [];
		if (withLayouts) {
			params.push("with_layouts=1")
		}
		if (sectionsOnly) {
			params.push("sections_only=1")
		}
		this.ajax({
			type: "GET",
			url: "/guides/resources?" + params.join("&"),
			dataType: "json"
		}, cb);
	},
	resourceGuidesImport: function(formData: any, cb: ajaxCB) {
		this.ajax({
			type: "PUT",
			cache: false,
			contentType: false,
			processData: false,
			url: "/guides/resources",
			data: formData
		}, cb);
	},
	resourceGuidesExport: function(cb: ajaxCB) {
		this.ajax({
			type: "GET",
			url: "/guides/resources?with_layouts=1&indented=1",
			dataType: "text"
		}, cb);
	},
	rxGuide: function(ndc: string, withHTML: bool, cb: ajaxCB) {
		var params = "";
		if (withHTML) {
			params = "?with_html=1"
		}
		this.ajax({
			type: "GET",
			url: "/guides/rx/" + ndc + params,
			dataType: "json"
		}, cb);
	},
	rxGuidesList: function(cb: ajaxCB) {
		this.ajax({
			type: "GET",
			url: "/guides/rx",
			dataType: "json"
		}, cb);
	},
	rxGuidesImport: function(formData: any, cb: ajaxCB) {
		this.ajax({
			type: "PUT",
			cache: false,
			contentType: false,
			processData: false,
			url: "/guides/rx",
			data: formData
		}, cb);
	},
	updateResourceGuide: function(id: string, guideUpdate: any, cb: ajaxCB) {
		this.ajax({
			type: "PATCH",
			contentType: "application/json",
			url: "/guides/resources/" + encodeURIComponent(id),
			data: JSON.stringify(guideUpdate),
			dataType: "json"
		}, cb);
	},
	createResourceGuide: function(guide: any, cb: ajaxCB) {
		this.ajax({
			type: "POST",
			contentType: "application/json",
			url: "/guides/resources",
			data: JSON.stringify(guide),
			dataType: "json"
		}, cb);
	},

	// Analytics

	analyticsQuery: function(q: string, cb: ajaxCB) {
		this.ajax({
			type: "POST",
			contentType: "application/json",
			url: "/analytics/query",
			data: JSON.stringify({query: q}),
			dataType: "json"
		}, cb);
	},
	listAnalyticsReports: function(cb: ajaxCB) {
		this.ajax({
			type: "GET",
			url: "/analytics/reports",
			dataType: "json"
		}, cb);
	},
	analyticsReport: function(id: string, cb: ajaxCB) {
		this.ajax({
			type: "GET",
			url: "/analytics/reports/" + encodeURIComponent(id),
			dataType: "json"
		}, cb);
	},
	runAnalyticsReport: function(id: string, cb: ajaxCB) {
		this.ajax({
			type: "POST",
			contentType: "application/json",
			url: "/analytics/reports/" + encodeURIComponent(id) + "/run",
			dataType: "json"
		}, cb);
	},
	createAnalyticsReport: function(name: string, query: string, presentation: string, cb: ajaxCB) {
		this.ajax({
			type: "POST",
			contentType: "application/json",
			url: "/analytics/reports",
			data: JSON.stringify({name: name, query: query, presentation: presentation}),
			dataType: "json"
		}, cb);
	},
	updateAnalyticsReport: function(id: string, name: string, query: string, presentation: string, cb: ajaxCB) {
		this.ajax({
			type: "POST",
			contentType: "application/json",
			url: "/analytics/reports/" + encodeURIComponent(id),
			data: JSON.stringify({name: name, query: query, presentation: presentation}),
			dataType: "json"
		}, cb);
	},

	// Admin accounts

	searchAdmins: function(query: string, cb: ajaxCB) {
		this.ajax({
			type: "GET",
			url: "/admins?q=" + encodeURIComponent(query),
			dataType: "json"
		}, cb);
	},
	adminAccount: function(id: string, cb: ajaxCB) {
		this.ajax({
			type: "GET",
			url: "/admins/" + encodeURIComponent(id),
			dataType: "json"
		}, cb);
	},
	adminPermissions: function(id: string, cb: ajaxCB) {
		this.ajax({
			type: "GET",
			url: "/admins/" + encodeURIComponent(id) + "/permissions",
			dataType: "json"
		}, cb);
	},
	adminGroups: function(id: string, cb: ajaxCB) {
		this.ajax({
			type: "GET",
			url: "/admins/" + encodeURIComponent(id) + "/groups",
			dataType: "json"
		}, cb);
	},
	updateAdminGroups: function(id: string, groups: any, cb: ajaxCB) {
		this.ajax({
			type: "POST",
			contentType: "application/json",
			url: "/admins/" + encodeURIComponent(id) + "/groups",
			data: JSON.stringify(groups),
			dataType: "json"
		}, cb);
	},
	availablePermissions: function(cb: ajaxCB) {
		this.ajax({
			type: "GET",
			url: "/accounts/permissions",
			dataType: "json"
		}, cb);
	},
	availableGroups: function(withPermissions: bool, cb: ajaxCB) {
		var params = "";
		if (withPermissions) {
			params = "?with_perms=1";
		}
		this.ajax({
			type: "GET",
			url: "/accounts/groups" + params,
			dataType: "json"
		}, cb);
	},

	// Drugs

	searchDrugs: function(query: string, cb: ajaxCB) {
		this.ajax({
			type: "GET",
			url: "/drugs?q=" + encodeURIComponent(query),
			dataType: "json"
		}, cb);
	},

	// Diagnoses

	searchDiagnosisCode: function(query: string, cb: ajaxCB)  {
		this.ajax({
			type: "GET",
			url: "/diagnosis/code?q=" + encodeURIComponent(query),
			dataType: "json"
		}, cb);
	},

	// Librato

	libratoQueryComposite: function(compose: string, resolution: number, startTime: number, endTime: ?number, count: ?number, cb: ajaxCB) {
		var query = "compose=" + encodeURIComponent(compose);
		query += "&resolution=" + encodeURIComponent(resolution.toString());
		query += "&start_time=" + encodeURIComponent(startTime.toString());
		if (endTime) {
			query += "&end_time=" + encodeURIComponent(endTime.toString());
		}
		if (count) {
			query += "&count=" + encodeURIComponent(count.toString());
		}
		this.ajax({
			type: "GET",
			url: "/librato/composite?" + query,
			dataType: "json"
		}, cb);
	},

	// Stripe

	stripeCharges: function(limit: ?number, cb: ajaxCB) {
		var query = "";
		if (limit) {
			query += "limit=" + encodeURIComponent(limit.toString());
		}
		this.ajax({
			type: "GET",
			url: "/stripe/charges?" + query,
			dataType: "json"
		}, cb);
	},

	// Pathways

	pathway: function(id: string, cb: ajaxCB) {
		this.ajax({
			type: "GET",
			url: "/pathways/" + encodeURIComponent(id),
			dataType: "json"
		}, cb);
	},
	pathways: function(activeOnly: bool, cb: ajaxCB) {
		var query = "";
		if (activeOnly) {
			query += "active_only=true";
		}
		this.ajax({
			type: "GET",
			url: "/pathways?" + query,
			dataType: "json"
		}, cb);
	},
	pathwayMenu: function(cb: ajaxCB) {
		this.ajax({
			type: "GET",
			url: "/pathways/menu",
			dataType: "json"
		}, cb);
	},
	createPathway: function(pathway: any, cb: ajaxCB) {
		this.ajax({
			type: "POST",
			contentType: "application/json",
			url: "/pathways",
			data: JSON.stringify({pathway: pathway}),
			dataType: "json"
		}, cb);
	},
	updatePathway: function(id: string, update: any, cb: ajaxCB) {
		this.ajax({
			type: "PATCH",
			contentType: "application/json",
			url: "/pathways/" + encodeURIComponent(id),
			data: JSON.stringify(update),
			dataType: "json"
		}, cb);
	},
	updatePathwayMenu: function(menu: any, cb: ajaxCB) {
		this.ajax({
			type: "PUT",
			contentType: "application/json",
			url: "/pathways/menu",
			data: JSON.stringify(menu),
			dataType: "json"
		}, cb);
	},
	diagnosisSets: function(pathwayTag: string, cb: ajaxCB) {
		this.ajax({
			type: "GET",
			url: "/pathways/diagnosis_sets?pathway_tag=" + pathwayTag,
			dataType: "json"
		}, cb);
	},
	updateDiagnosisSet: function(pathwayTag: string, update: any, cb: ajaxCB) {
		this.ajax({
			type: "PATCH",
			contentType: "application/json",
			url: "/pathways/diagnosis_sets",
			data: JSON.stringify(update),
			dataType: "json"
		}, cb);
	},

	// Templates
	layoutVersions: function(cb: ajaxCB) {
		this.ajax({
			type: "GET",
			url: "/layouts/version",
			dataType: "json"
		}, cb);
	},
	question: function(tag: string, language_id: string, version: ?number, cb: ajaxCB, async: bool) {
		version = version == null ? 1 : version
		var query = "tag="+encodeURIComponent(tag)+"&version="+encodeURIComponent(version.toString())+"&language_id="+encodeURIComponent(language_id)
		this.ajax({
			type: "GET",
			url: "/layouts/versioned_question?" + query,
			dataType: "json"
		}, cb, async);
	},
	template: function(pathway_tag: string, sku: string, purpose: string, major: string, minor: string, patch: any, cb: ajaxCB) {
		var query = "pathway_tag="+encodeURIComponent(pathway_tag)+"&sku="+encodeURIComponent(sku)+"&purpose="+encodeURIComponent(purpose)+"&major="+encodeURIComponent(major)+"&minor="+encodeURIComponent(minor)+"&patch="+encodeURIComponent(patch)
		this.ajax({
			type: "GET",
			url: "/layouts/template?" + query,
			dataType: "json"
		}, cb);
	},
	layoutUpload: function(formData: any, cb: ajaxCB, async: bool) {
		this.ajax({
			type: "POST",
			cache: false,
			contentType: false,
			processData: false,
			url: "/layout",
			data: formData
		}, cb, async);
	},
	submitQuestion: function(question: any, cb: ajaxCB, async: bool) {
		this.ajax({
			type: "POST",
			contentType: "application/json",
			url: "/layouts/versioned_question",
			data: JSON.stringify(question),
			dataType: "json"
		}, cb, async);
	},

	// STP
	sampleTreatmentPlan: function(pathway: string, cb: ajaxCB) {
		var query = "pathway_tag="+encodeURIComponent(pathway)
		this.ajax({
			type: "GET",
			url: "/sample_treatment_plan?" + query,
			dataType: "json"
		}, cb);
	},
	updateSampleTreatmentPlan: function(pathway: string, stp: any, cb: ajaxCB) {
		this.ajax({
			type: "PUT",
			contentType: "application/json",
			url: "/sample_treatment_plan",
			data: JSON.stringify({pathway_tag: pathway, sample_treatment_plan: stp}),
			dataType: "json"
		}, cb);
	},

	// Financial
	incomingFinancialItems: function(from: string, to: string, cb: ajaxCB) {
		var query = "from=" + encodeURIComponent(from) + "&to=" + encodeURIComponent(to);
		this.ajax({
			type: "GET",
			url: "/financial/incoming?" + query,
			dataType: "json"
		}, cb);
	},
	outgoingFinancialItems: function(from: string, to: string, cb: ajaxCB) {
		var query = "from=" + encodeURIComponent(from) + "&to=" + encodeURIComponent(to);
		this.ajax({
			type: "GET",
			url: "/financial/outgoing?"+query,
			dataType: "json"
		}, cb);
	},

	visitSKUs: function(activeOnly: bool, cb: ajaxCB) {
		this.ajax({
			type: "GET",
			url: "/financial/skus/visit?active_only="+encodeURIComponent(activeOnly.toString()),
			dataType: "json"
		}, cb);
	},

	// FTP Interaction
	globalFavoriteTreatmentPlans: function(cb: ajaxCB) {
		this.ajax({
			type: "GET",
			url: "/treatment_plan/favorite/global",
			dataType: "json"
		}, cb);
	},
	favoriteTreatmentPlans: function(id: string, cb: ajaxCB) {
		this.ajax({
			type: "GET",
			url: "/treatment_plan/favorite/" + encodeURIComponent(id),
			dataType: "json"
		}, cb);
	},
	favoriteTreatmentPlanMemberships: function(id: string, cb: ajaxCB) {
		this.ajax({
			type: "GET",
			url: "/treatment_plan/favorite/" + encodeURIComponent(id) + "/membership",
			dataType: "json"
		}, cb);
	},
	createFavoriteTreatmentPlanMemberships: function(ftpID: string, memberships: any, cb: ajaxCB) {
		var body = {requests: memberships}
		this.ajax({
			type: "POST",
			contentType: "application/json",
			url: "/treatment_plan/favorite/" + encodeURIComponent(ftpID) + "/membership",
			data: JSON.stringify(body),
			dataType: "json"
		}, cb);
	},
	deleteFavoriteTreatmentPlanMembership: function(ftpID: string, doctorID: string, pathwayTag: string, cb: ajaxCB) {
		this.ajax({
			type: "DELETE",
			contentType: "application/json",
			url: "/treatment_plan/favorite/" + encodeURIComponent(ftpID) + "/membership",
			data: JSON.stringify({doctor_id: doctorID, pathway_tag: pathwayTag}),
			dataType: "json"
		}, cb);
	},

	// SAML APIs
	transformSAML: function(saml: string, cb: ajaxCB) {
		this.ajax({
			type: "POST",
			contentType: "application/json",
			url: "/layout/saml",
			data: JSON.stringify({saml: saml}),
			dataType: "json"
		}, cb);
	},

	// Case/Visit Interaction
	visitSummaries: function(visitStatus: string, cb: ajaxCB) {
		var query = "status=" + encodeURIComponent(visitStatus)
		this.ajax({
			type: "GET",
			contentType: "application/json",
			url: "/case/visit?"+query,
			dataType: "json"
		}, cb);
	},
	visitSummary: function(caseID: string, visitID: string, cb: ajaxCB) {
		this.ajax({
			type: "GET",
			contentType: "application/json",
			url: "/case/" + encodeURIComponent(caseID) + "/visit/" + encodeURIComponent(visitID),
			dataType: "json"
		}, cb);
	},
	visitEvents: function(visitID: string, caseID: string, cb: ajaxCB) {
		if (visitID && caseID) {
			this.ajax({
				type: "GET",
				contentType: "application/json",
				url: "/event/server?visit_id=" + encodeURIComponent(visitID) + "&case_id=" + encodeURIComponent(caseID),
				dataType: "json"
			}, cb);
		} else {
			console.error("Both visitID and caseID are expected")
			cb(false, null, {message:"Both visitID and caseID are expected"})
		}
	},

	// Dynamic config
	cfg: function(cb: ajaxCB) {
		this.ajax({
			type: "GET",
			contentType: "application/json",
			url: "/cfg",
			dataType: "json"
		}, cb);
	},
	updateCfg: function(update: any, cb: ajaxCB) {
		this.ajax({
			type: "PATCH",
			contentType: "application/json",
			url: "/cfg",
			data: JSON.stringify({"values": update}),
			dataType: "json"
		}, cb);
	},

	// Scheduled message templates
	listScheduledMessageTemplates: function(cb: ajaxCB) {
		this.ajax({
			type: "GET",
			contentType: "application/json",
			url: "/schedmsgs/templates",
			dataType: "json"
		}, cb);
	},
	scheduledMessageTemplate: function(id: string, cb: ajaxCB) {
		this.ajax({
			type: "GET",
			contentType: "application/json",
			url: "/schedmsgs/templates/" + encodeURIComponent(id),
			dataType: "json"
		}, cb);
	},
	updateScheduledMessageTemplate: function(id: string, template: any, cb: ajaxCB) {
		this.ajax({
			type: "PUT",
			contentType: "application/json",
			url: "/schedmsgs/templates/" + encodeURIComponent(id),
			data: JSON.stringify(template),
			dataType: "json"
		}, cb);
	},

	// Tagging
	tags: function(text: string, common: bool, cb: ajaxCB) {
		this.ajax({
			type: "GET",
			contentType: "application/json",
			url: "/tag?text=" + encodeURIComponent(text) + "&common=" + encodeURIComponent(common.toString()),
			dataType: "json"
		}, cb);
	},
	addTag: function(tagText: string, common: bool, cb: ajaxCB) {
		this.ajax({
			type: "POST",
			contentType: "application/json",
			url: "/tag",
			data: JSON.stringify({text: tagText, common: common}),
			dataType: "json"
		}, cb);
	},
	updateTag: function(tagID: number, common: bool, cb: ajaxCB) {
		this.ajax({
			type: "PUT",
			contentType: "application/json",
			url: "/tag",
			data: JSON.stringify({id: tagID, common: common}),
			dataType: "json"
		}, cb);
	},
	savedTagSearches: function(cb: ajaxCB) {
		this.ajax({
			type: "GET",
			contentType: "application/json",
			url: "/tag/saved_search",
			dataType: "json"
		}, cb);
	},
	addTagSearch: function(title: string, query: string, cb: ajaxCB) {
		this.ajax({
			type: "POST",
			contentType: "application/json",
			url: "/tag/saved_search",
			data: JSON.stringify({title: title, query: query}),
			dataType: "json"
		}, cb);
	},
	deleteTagSearch: function(id: number, cb: ajaxCB) {
		this.ajax({
			type: "DELETE",
			contentType: "application/json",
			url: "/tag/saved_search/"+ encodeURIComponent(id.toString()),
			dataType: "json"
		}, cb);
	},

	// Email

	sendTestEmail: function(type: string, cb: ajaxCB) {
		this.ajax({
			type: "POST",
			contentType: "application/json",
			url: "/email/test",
			data: JSON.stringify({type: type}),
			dataType: "json"
		}, cb);
	},
};

var noError: ajaxError = {message: ""};

function parseError(jqXHR: jqXHR): ajaxError {
	if (jqXHR.status == 0) {
		return {message: "Network request failed"};
	}
	var err;
	try {
		err = JSON.parse(jqXHR.responseText).error;
	} catch(e) {
		if (jqXHR.status == 403) {
			err = {message: "Access denied"};
		} else {
			console.error(jqXHR.responseText);
			err = {message: "Unknown error"};
		}
	}
	return err;
}
