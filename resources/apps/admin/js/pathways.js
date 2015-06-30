/* @flow */

var AdminAPI = require("./api.js");
var IntakeReview = require("./intake_review.js");
var Forms = require("../../libs/forms.js");
var jsyaml = require("js-yaml");
var Modals = require("../../libs/modals.js");
var Nav = require("../../libs/nav.js");
var Perms = require("./permissions.js");
var React = require("react/addons");
var Routing = require("../../libs/routing.js");
var Utils = require("../../libs/utils.js");
var Financial = require("./financial.js");
var $ = require("jquery");

var ReactCSSTransitionGroup = React.addons.CSSTransitionGroup;

module.exports = {
	Page: React.createClass({displayName: "PathwaysPage",
		menuItems: [[
			{
				id: "list",
				url: "/admin/pathways",
				name: "Pathways"
			},
			{
				id: "menu",
				url: "/admin/pathways/menu",
				name: "Menu"
			},
			{
				id: "intake_templates",
				url: "/admin/pathways/intake_templates",
				name: "Intake Templates"
			},
			{
				id: "diagnosis_sets",
				url: "/admin/pathways/diagnosis_sets",
				name: "Diagnosis Sets"
			},
			{
				id: "global_favorite_treatment_plans",
				url: "/admin/pathways/global_favorite_treatment_plans",
				name: "Global Favorite Treatment Plans"
			},
			{
				id: "saml",
				url: "/admin/pathways/saml",
				name: "SAML"
			}
		]],
		pages: {
			list: function(): any {
				return <ListPage router={this.props.router} />;
			},
			menu: function(): any {
				return <MenuPage router={this.props.router} />;
			},
			intake_templates: function(): any {
				return <IntakeTemplatesPage router={this.props.router} />;
			},
			details: function(): any {
				return <DetailsPage router={this.props.router} pathwayID={this.props.pathwayID} />;
			},
			diagnosis_sets: function(): any {
				return <DiagnosisSets router={this.props.router} />;
			},
			global_favorite_treatment_plans: function(): any {
				return <GlobalFTPPage router={this.props.router} />;
			},
			saml: function(): any {
				return <SAMLPage router={this.props.router} />;
			},
		},
		getDefaultProps: function(): any {
			return {}
		},
		render: function(): any {
			return (
				<div>
					<Nav.LeftNav router={this.props.router} items={this.menuItems} currentPage={this.props.page}>
						{this.pages[this.props.page].bind(this)()}
					</Nav.LeftNav>
				</div>
			);
		}
	})
};

var GlobalFTPPage = React.createClass({displayName: "GlobalFTPPage",
	mixins: [Routing.RouterNavigateMixin],
	getInitialState: function(): any {
		return {
			ftps: [],
			ftp_fetch_error: null
		};
	},
	componentWillMount: function() {
		document.title = "Global Favorite Treatment Plans";
		AdminAPI.globalFavoriteTreatmentPlans(function(success, data, error) {
			if (this.isMounted()) {
				if (!success) {
					this.setState({ftp_fetch_error: error.message});
					return;
				}
				data.favorite_treatment_plans.sort(function(a, b){
					if(a.name < b.name) return -1
					if(a.name > b.name) return 1
					return 0
				})
				this.setState({
					ftps: data.favorite_treatment_plans,
					ftp_fetch_error: null
				});
			}
		}.bind(this));
	},
	render: function(): any {
		var content = []
		for(var i = 0; i < this.state.ftps.length; ++i) {
			content.push(<tr><td>
												<a href={"/admin/treatment_plan/favorite/" + this.state.ftps[i].id + "/info"} onClick={this.onNavigate}>
													{this.state.ftps[i].name}
												</a>
											</td></tr>)
		}
		return (
			<div className="container" style={{marginTop: 10}}>
				<div className="row">
					<div className="col-sm-12 col-md-12 col-lg-9">
						<h2>Global Favorite Treatment Plans</h2>
					</div>
				</div>

				<div className="row">
					<div className="col-md-12">
						<div className="text-center">
							{this.state.busy ? <Utils.LoadingAnimation /> : null}
							{this.state.ftp_fetch_error ? <Utils.Alert type="danger">{this.state.ftp_fetch_error}</Utils.Alert> : null}
						</div>
					</div>
				</div>

				<div className="row">
					<div className="col-md-12">
						<table className="table">
							<thead>
								<tr>
									<th>Name</th>
								</tr>
							</thead>
							<tbody>
								{content}
							</tbody>
						</table>
					</div>
				</div>
			</div>

		);
	}
});


var DiagnosisSets = React.createClass({displayName: "DiagnosisSets",
	mixins: [Routing.RouterNavigateMixin],
	getInitialState: function(): any {
		return {
			pathwayTag: "",
			pathwayName: "",
			diagnosisSets: null,
			busy: false
		};
	},
	componentWillMount: function() {
		document.title = "Pathways | Diagnosis Sets";
		this.setState({busy: false});
	},
	componentDidMount: function() {
		this.setState({busy:false, pathwayTag: "health_condition_acne", pathwayName: "Acne"});
	},
	handlePathwayChange: function(e: any, pathwayTag: string, pathwayName: string): void {
		e.preventDefault();
		this.setState({pathwayTag: e.target.value, pathwayName: pathwayName});
	},
	render: function(): any {
		var content;
		if (this.state.pathwayTag != "") {
			content = <PathwayDiagnosisSet
				pathwayTag = {this.state.pathwayTag}
				pathwayName = {this.state.pathwayName} />
		}
		return (
			<div className="container" style={{marginTop: 10}}>
				<div className="row">
					<div className="col-sm-12 col-md-12 col-lg-9">
						<h2>Diagnosis Sets</h2>
						<AvailablePathwaysSelect onChange={this.handlePathwayChange} />
					</div>
				</div>

				<div className="row">
					<div className="col-md-12">
						<div className="text-center">
							{this.state.busy ? <Utils.LoadingAnimation /> : null}
							{this.state.error ? <Utils.Alert type="danger">{this.state.error}</Utils.Alert> : null}
						</div>
					</div>
				</div>

				<div className="row">
					<div className="col-md-12">
						{content}
					</div>
				</div>
			</div>
		);
	}
});


var PathwayDiagnosisSet = React.createClass({displayName: "PathwayDiagnosisSet",
	mixins: [Routing.RouterNavigateMixin],
	getInitialState: function(): any {
		return {
			error: null,
			busy: false,
			title: "",
			items: [],
			deletes: {},
			creates: []
		}
	},
	componentDidMount: function() {
		this.setState({title: "Common " + this.props.pathwayName + " Diagnoses"})
		this.fetchDiagnosisSet(this.props.pathwayTag);
	},
	componentWillMount: function() {
		this.fetchDiagnosisSet(this.props.pathwayTag);
	},
	componentWillReceiveProps: function(nextProps) {
		this.fetchDiagnosisSet(nextProps.pathwayTag);
	},
	reset: function() {
		this.setState({
			deletes: {},
			creates: [],
		});
	},
	fetchDiagnosisSet: function(pathwayTag: string) {
		this.setState({busy: true, error: null});
		AdminAPI.diagnosisSets(pathwayTag,
			function(success, res, error) {
				if (this.isMounted()) {
					if (success) {
						if (res.title != "") {
							this.state.title = res.title
						} else {
							this.state.title = "Common " + this.props.pathwayName + " Diagnoses"
						}
						this.setState({busy: false, title: this.state.title, items: res.items});
					} else {
						this.setState({busy: false, error: error.message, title: "", items: []});
					}
				}
			}.bind(this));
	},
	handleDelete: function(codeID: string, e: Event) {
		e.preventDefault();
		this.state.deletes[codeID] = !(this.state.deletes[codeID] || false);
		this.setState({deletes: this.state.deletes});
	},
	handleAdd: function(codeID: string, code: string, description: string) {
		this.state.creates.push({codeID: codeID, code: code, name: description});
		this.setState({creates: this.state.creates});
	},
	handleCancelAdd: function(index, e) {
		e.preventDefault();
		for(var i = index; i < this.state.creates.length-1; i++) {
			this.state.creates[i] = this.state.creates[i+1];
		}
		this.state.creates.pop();
		this.setState({creates: this.state.creates});
	},
	handleCancel: function(e) {
		e.preventDefault();
		this.reset();
	},
	handleSave: function(e) {
		e.preventDefault();
		this.setState({busy: true, error: ""});
		var patch = {
			"delete": [],
			"title": this.state.title,
			"create": [],
			"pathwayTag": this.props.pathwayTag
		};
		for (var codeID in this.state.deletes) {
			if (this.state.deletes[codeID]) {
				patch.delete.push(codeID);
			}
		}
		this.state.creates.forEach(function(dsi) {
			patch.create.push(dsi.codeID);
		});
		AdminAPI.updateDiagnosisSet(this.props.pathwayTag, patch, function(success, res, error){
			if (success) {
				if (this.isMounted()) {
					this.reset();
					this.setState({
						items: res.items,
						busy: false
					});
				}
			} else {
				this.setState({busy: false, error: error.message});
			}
		}.bind(this));
	},
	render: function () {
		var existingCodes = {}
		{this.state.items.map(function(dsi) {
			existingCodes[dsi.code] = true;
		}.bind(this))};
		{this.state.creates.map(function(dsi){
			existingCodes[dsi.code] = true;
		}.bind(this))};

		return (
			<div>
				{Perms.has(Perms.PathwaysEdit) ? <AddDiagnosisModal existingDiagnosisCodes={existingCodes} onSuccess={this.handleAdd} /> : null}
				<div className="text-left">
					{this.state.busy ? <Utils.LoadingAnimation /> : null}
					{this.state.title != "" && Perms.has(Perms.PathwaysEdit) ?
						<div className="pull-right"><button className="btn btn-default" data-toggle="modal" data-target="#add-diagnosis-modal">+</button></div>
						: null}
					{this.state.title != "" ? <h3> {this.state.title} </h3> : null}
				</div>

				<table className="table">
				<thead>
					<tr>
						<th>Code</th>
						<th>Name</th>
					</tr>
				</thead>
				<tbody>
					{this.state.items.map(function(dsi) {
						return (
							<tr key={"item-"+dsi.codeID}
								style={
									this.state.deletes[dsi.codeID] === true ? {
										textDecoration: "line-through",
										backgroundColor: "#ffa0a0"
									} : {}}
							>
								<td>{dsi.code}</td>
								<td>{dsi.name}</td>
								{Perms.has(Perms.PathwaysEdit) ?
										<td>
											<a href="#" onClick={this.handleDelete.bind(this, dsi.codeID)}>
												<span className="glyphicon glyphicon-remove" style={{color:"red"}}></span>
											</a>
										</td>
									: null}
							</tr>
						);
					}.bind(this))}
					{this.state.creates.map(function(dsi, index) {
							return (
								<tr key={"new-diagnosis-" + index} style={{backgroundColor: "#a0ffa0"}}>
									<td>{dsi.code}</td>
									<td>{dsi.name}</td>
									{Perms.has(Perms.PathwaysEdit) ?
										<td>
											<a href="#" onClick={this.handleCancelAdd.bind(this, index)}>
												<span className="glyphicon glyphicon-remove" style={{color:"red"}}></span>
											</a>
										</td>
									: null}
								</tr>
							);
						}.bind(this))}
				</tbody>
				</table>

				<div className="text-center">
					{this.state.error ? <Utils.Alert type="danger">{this.state.error}</Utils.Alert> : null}
				</div>

				{Perms.has(Perms.PathwaysEdit) ?
					<div className="text-right">
						<button className="btn btn-default" onClick={this.handleCancel}>Cancel</button>
						{" "}<button className="btn btn-primary" onClick={this.handleSave}>Save</button>
					</div>
				: null}


			</div>
		);
	}
});

var AddDiagnosisModal = React.createClass({displayName: "AddDiagnosisModal",
	mixins: [Routing.RouterNavigateMixin],
	getInitialState: function(): any {
		return {
			error: "",
			busy: false,
			icd10Code: ""
		};
	},
	onAdd: function(e) {

		if (this.props.existingDiagnosisCodes[this.state.icd10Code]) {
			this.setState({error: "code already exists in set"})
			return true;
		}

		this.setState({busy: true, error: ""})

		// search to ensure that the diagnosis code entered
		// truly is a diagnosis code. Don't accept if more
		// than one result returned
		AdminAPI.searchDiagnosisCode(this.state.icd10Code,
			function(success, res, error) {
				if (this.isMounted()) {

					if (!success) {
						this.setState({busy: false, error: error.message})
						return;
					}

					if (res.results.length > 1) {
						this.setState({busy: false, error:"more than one diagnosis returned"})
						return;
					}

					this.setState({busy: false});
					this.props.onSuccess(res.results[0].codeID, res.results[0].code, res.results[0].description);
					$("#add-diagnosis-modal").modal('hide');
				}

				}.bind(this));
		return true;
	},
	onChangeICD10Code: function(e) {
		e.preventDefault();
		this.setState({error: "", icd10Code: e.target.value});
	},
	render: function(): any {
		return (
			<Modals.ModalForm id="add-diagnosis-modal" title="Add Diagnosis"
				cancelButtonTitle="Cancel" submitButtonTitle="Add"
				onSubmit={this.onAdd}>

				{this.state.error ? <Utils.Alert type="danger">{this.state.error}</Utils.Alert> : null}

				<Forms.FormInput label="Add ICD10 Code" value={this.state.icd10Code} onChange={this.onChangeICD10Code} />
			</Modals.ModalForm>
		);
	}
});


var IntakeTemplatesPage = React.createClass({displayName: "IntakeTemplatesPage",
	mixins: [Routing.RouterNavigateMixin],
	getInitialState: function(): any {
		var yamlString = jsyaml.safeDump(this.intake_spec)
		return {
			intake_json: yamlString,
			review_json: "",
			busy: false,
			error: null,
			visitSKU: "",
			review_versions: [],
			intake_versions: []
		};
	},
	componentWillMount: function() {
		document.title = "Pathways | Intake Templates";
		this.setState({busy: false});
	},
	componentDidMount: function() {
		this.setState({busy: true});
		AdminAPI.visitSKUs(true, function(success, data, error) {
			var skus = []
			for(var i = 0; i < data.skus.length; i++) {
				skus.push(data.skus[i])
			}
			this.setState({
					busy: false,
					visitSKU: skus[0]
				})
			this.updatePathwayVersions(skus[0])
		}.bind(this));
	},
	onChangeIntake: function(e: any) {
		e.preventDefault();
		var error = null;
		try {
			jsyaml.safeLoad(e.target.value)
		} catch(ex) {
			error =  e.message ? "Invalid YAML: " + ex.message : ex
		}
		this.setState({
			intake_error: error,
			intake_json: e.target.value
		});
	},
	onChangeReview: function(e: any) {
		e.preventDefault();
		var error = null;
		try {
			jsyaml.safeLoad(e.target.value)
		} catch(ex) {
			error = e.message ? "Invalid YAML: " + ex.message : ex
		}
		this.setState({
			review_error: error,
			review_json: e.target.value
		});
	},
	onVisitSKUChange: function(e: Event, sku: string) {
		this.updatePathwayVersions(sku)
	},
	updatePathwayVersions: function(sku: string) {
		AdminAPI.layoutVersions(function(success, data, error) {
			var intake_versions = [];
			var review_versions = [];

			var items = data.items
			for (var i = 0; i < items.length; i++) {
				if (items[i].SKUType == sku) {                  
					if (items[i].LayoutPurpose == "CONDITION_INTAKE") {
						intake_versions.push(items[i].Version)  
					} else if (items[i].LayoutPurpose == "REVIEW") {
						review_versions.push(items[i].Version)
					}
				}
			}

			var newest_intake_version = intake_versions[intake_versions.length - 1]
			var newest_review_version = review_versions[review_versions.length - 1]

			var intake_json = intake_versions.length == 0 ? jsyaml.safeDump(this.intake_spec) : this.state.intake_json
			this.setState({
				visitSKU: sku,
				intake_json: intake_json,
				intake_versions: intake_versions,
				newest_intake_version: newest_intake_version,
				review_versions: review_versions,
				newest_review_version: newest_review_version,
			});
		}.bind(this));
	},
	onIntakeVersionSelection: function(version: string) {
		var versionParts = version.split(".")
		var pathway_tag = IntakeReview.pathwayFromSKU(this.state.visitSKU)
		AdminAPI.template(pathway_tag, this.state.visitSKU, "CONDITION_INTAKE", versionParts[0], versionParts[1], versionParts[2], function(success, data, error) {
			try {
				data = IntakeReview.expandTemplate(data, this.updateIntakeInfo)
				var yamlString = jsyaml.safeDump(data)
				this.setState({
					intake_json: yamlString,
					intake_info: null
				});
			} catch (ex) {
				this.setState({
					intake_error:  ex.message ? ex.message : ex,
				});
			}
		}.bind(this));
	},
	onReviewVersionSelection: function(version: string) {
		var versionParts = version.split(".")
		var pathwayTag = IntakeReview.pathwayFromSKU(this.state.visitSKU)
		AdminAPI.template(pathwayTag, this.state.visitSKU, "REVIEW", versionParts[0], versionParts[1], versionParts[2], function(success, data, error) {
			var yamlString = jsyaml.safeDump(data)
			this.setState({
				review_json: yamlString,
			});
		}.bind(this));
	},
	generateReview: function(e: Event) {
		e.preventDefault();
		try {
			var pathwayTag = IntakeReview.pathwayFromSKU(this.state.visitSKU)
			var review = IntakeReview.generateReview(jsyaml.safeLoad(this.state.intake_json), pathwayTag)
			this.setState({
				review_json: jsyaml.safeDump(review),
				review_error: null,
			});
		} catch (ex) {
			this.setState({
				intake_error: ex.message ? ex.message : ex,
			});
		}
	},
	submitLayout: function(e: Event) {
		e.preventDefault();
		this.setState({busy: true});
		var intake = jsyaml.safeLoad(this.state.intake_json)
		var review = jsyaml.safeLoad(this.state.review_json)

		// start with layout major version 3 to account for the previous 2 major versions that we have supported before using this tool.
		var intake_v = this.state.newest_intake_version != undefined ? this.state.newest_intake_version.split(".") : ["3","-1","0"]
		var review_v = this.state.newest_review_version != undefined ? this.state.newest_review_version.split(".") : ["3","-1","0"]

		// if we are upgrading a version 1 or 2 layout, upgrade it to major version 3 for the same reason above.
		if (intake_v[0] == "1" || intake_v[0] == "2") {
			intake_v[0] = "3"
		}
		if (review_v[0] == "1" || review_v[0] == "2") {
			review_v[0] = "3"
		}

		intake.version = intake_v[0] + "." + (parseInt(intake_v[1]) + 1) + "." + intake_v[2]
		review.version = review_v[0] + "." + (parseInt(review_v[1]) + 1) + "." + review_v[2]
		try {
			var pathwayTag = IntakeReview.pathwayFromSKU(this.state.visitSKU)
			intake.cost_item_type = this.state.visitSKU
			IntakeReview.submitLayout(intake, review, pathwayTag, this.updateSubmitInfo)
			this.setState({
				success_text: "Intake " + intake.version + " and Review " + review.version + " created for Pathway " + pathwayTag + " and SKU " + this.state.visitSKU,
				submit_info: null,
				review_error: null,
				busy: false,
			})
			this.updatePathwayVersions(this.state.visitSKU)
		} catch (ex) {
			this.setState({
				success_text: null,
				review_error:  ex.message ? ex.message : ex,
				busy: false
			})
		}
	},
	updateIntakeInfo: function(msg: string) {
		this.setState({
			intake_info: msg
		})
	},
	updateSubmitInfo: function(msg: string) {
		this.setState({
			submit_info: msg
		})
	},
	render: function(): any {
		return (
			<div>
				<div className="row">
					<div className="col-sm-12 col-md-12 col-lg-9">
						<h2>Pathway Templates</h2>
						<Financial.AvailableVisitSKUsSelect onChange={this.onVisitSKUChange} />
						{this.state.intake_json ?
							<form role="form" onSubmit={this.onSubmit} method="PUT">
								<div>
									{Perms.has(Perms.LayoutEdit) ?
										<Forms.TextArea name="json" required label="Intake Template" value={this.state.intake_json} rows="20" onChange={this.onChangeIntake} tabs={true} />
									: <pre>{this.state.intake_json}</pre>}
									{this.state.intake_error ? <Utils.Alert type="danger">{this.state.intake_error}</Utils.Alert> : null}
									{this.state.intake_info? <Utils.Alert type="info">{this.state.intake_info}</Utils.Alert> : null}
								</div>
								<button className="btn" href="#" onClick={this.generateReview}>Generate Review</button>
								<div>
									{Perms.has(Perms.LayoutEdit) ?
										<Forms.TextArea name="json" required label="Review Template" value={this.state.review_json} rows="20" onChange={this.onChangeReview} tabs={true} />
									: <pre>{this.state.review_json}</pre>}
									{this.state.review_error ? <Utils.Alert type="danger">{this.state.review_error}</Utils.Alert> : null}
									{this.state.submit_info? <Utils.Alert type="info">{this.state.submit_info}</Utils.Alert> : null}
									{this.state.success_text ? <Utils.Alert type="success">{this.state.success_text}</Utils.Alert> : null}
								</div>
								<div className="text-left">
									{this.state.busy ? <Utils.LoadingAnimation /> : null}
									{Perms.has(Perms.LayoutEdit) ?
										<button type="submit" onClick={this.submitLayout} className="btn btn-primary">Save</button>
									:null}
								</div>
							</form>
						:
							<div>
								{this.state.busy ? <Utils.LoadingAnimation /> : null}
							</div>
						}
					</div>
					<div className="col-sm-12 col-md-12 col-lg-3">
						{ this.state.intake_versions.length != 0 ? <AvailableIntakeTemplatesList intake_versions={this.state.intake_versions} onClick={this.onIntakeVersionSelection}/>: "" }
						{ this.state.intake_versions.length != 0 ? <AvailableReviewTemplatesList review_versions={this.state.review_versions} onClick={this.onReviewVersionSelection}/> : "" }
					</div>
				</div>
			</div>
		);
	},
	intake_spec: {
		"sections": [
			{
				"screens": [
					{
						"auto|section": "An identifier for the section - If not provided one will be generated",
							"auto|section_id":   "The new identifier for the section - If not provided the `section` attribute will be use",
							"section_title": "The section title to be presented to the client",
							"transition_to_message": "The message to display to the user when transitioning into this section",
						"questions": [
							{
								"optional|condition" : {
											"op": "answer_equals_exact | answer_contains_any | answer_contains_all | gender_equals | and | or",
											"*gender" : "male|female", // Required if gender_equals is the op
											"*operands" : [{ // Required if selected operation is and | or
												"op" : "answer_equals_exact | answer_contains_any | answer_contains_all | gender_equals | and | or",
												// this is a recursive definition of a condition object
											}],
											"*auto|question_tag": "The tag of the question that you are referencing in this conditp®ional", // Required if the selected 'op' is answer_xxxxx
											"*answer_tags": ["List of the answer tags to evaluate in this conditional"] // Required if the selected 'op' is answer_xxxxx
										},

								"details": {
									"auto|required": true, // true|false - representing if this question is required to be answered by the user
									"auto|unique|tag": "Generated if not specified. Should be specified if referenced elsewhere. Will have global|pathway_tag prepended",
									"auto|text_has_tokens": false, // true|false - representing if this string used tokens,
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

											"text": "The literal answer text shown to the user",
										},
										{
											// Other question answers
										}
									],
									"auto|photo_slots": [
									  {
										  "optional|type": "The type of photo slot to be presented to the user",
										  "optional|client_data": "The data to send to the client to aid in creation of this view",
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
									"optional|screen": [], // Must contain a screen object, parent question must be a q_type_multiple_choice question. Generally used with header title tokens or question title tokens that allow the parent answer text to be inserted into the header title or question title.
									"optional|questions": [] // Parent question must be a q_type_autocomplete - Don't use otherwise. Contains an array of question objects.
								}
							}
						],
						"screen_type": "screen_type_photo",
						"optional|subtitle": "Your doctor will use these photos to make a diagnosis."
					},
					{
						"screen_type": "screen_type_pharmacy"
					}
				]
			}
		]
	}
});

var PreviewBackButtonRow = React.createClass({
	render: function() {
		return <div className="pathways_menu_preview_back_button_label">&lt; Back</div>;
	}
});

var PreviewSubmenuRow = React.createClass({
	render: function() {
		return <div className="pathways_menu_preview_label">{this.props.menuItem.title}</div>;
	}
});

var PreviewPathwayRow = React.createClass({
	render: function() {
		return <div className="pathways_menu_preview_label">{this.props.pathwayItem.title}</div>;
	}
});

var PreviewMenu = React.createClass({
	handlePathwayClick: function(pathway) {
		// TODO: present Pathway Details
	},
	handleSubmenuClick: function(menu) {
		this.props.handleSubmenuClick(menu);
	},
	handleBackButtonClick: function() {
		this.props.handleBackButtonClick();
	},
	render: function() { 
		var backButtonStyle = {
			cursor: 'pointer'
		};
		var backButton;
		if (!this.props.isTopLevel) {
			backButton = (
				<div 
					key="pathways_menu_preview_back_button_row" 
					style={backButtonStyle}
					onClick={this.handleBackButtonClick}
				>
					<PreviewBackButtonRow />
				</div>
			);
		}
		var rows = [];
		this.props.menu.items.forEach(function(item) {
			if (item.pathway_tag) {
				rows.push(
					<div 
						key={item.pathway_tag} 
						className="pathways_menu_preview_pathway_row"
						onClick={this.handlePathwayClick.bind(this, item.pathway_tag)}
					>
						<PreviewPathwayRow pathwayItem={item} />
					</div>
				);
			} else {
				rows.push(
					<div 
						key={item.title} 
						className="pathways_menu_preview_submenu_row"
						onClick={this.handleSubmenuClick.bind(this, item)}
					>
						<PreviewSubmenuRow 
							onSubmenuClick={this.handleSubmenuClick}
							menuItem={item}
						/>
					</div>
				);
			}
		}.bind(this));


		var pathwaysMenuPreviewStyle = {
			zIndex: this.props.zIndex,
			position: 'absolute',
			width: '320px',
			height: '568px',
			padding: '16px 16px',
			overflow: 'scroll'
		};
		return (
			<div style={pathwaysMenuPreviewStyle}>
				{backButton}
				<div className="pathways_menu_preview_title">
					{this.props.menu.title}
				</div>
				<div>
					{rows}
				</div>
			</div>
		);
	}
});

var PreviewMenuContainer = React.createClass({
	getInitialState: function() {
		return {
			menuStack: []
		};
	},

	handleSubmenuClick: function(submenuItemToPresent) {
		var newMenuStack = this.state.menuStack.slice();
		newMenuStack.unshift(submenuItemToPresent.menu);
		this.setState({
			menuStack: newMenuStack
		});
	},
	handleBackButtonClick: function() {
		var newMenuStack = this.state.menuStack.slice();
		newMenuStack.shift();
		this.setState({
			menuStack: newMenuStack
		});
	},
	render: function() {
		if (this.state.menuStack[this.state.menuStack.length - 1] !== this.props.topLevelMenu) {
			this.state.menuStack = [this.props.topLevelMenu];
		}
		return (
			<ReactCSSTransitionGroup transitionName="animatable_preview_menu" key="animatable_preview_menu_container">
				<PreviewMenu 
					key={this.state.menuStack[0].title}
					menu={this.state.menuStack[0]}
					isTopLevel={this.state.menuStack.length === 1 ? true : false}
					handleSubmenuClick={this.handleSubmenuClick}
					handleBackButtonClick={this.handleBackButtonClick}
					zIndex={this.state.menuStack.length}
				/>
			</ReactCSSTransitionGroup>
		);
	}
});

var MenuPage = React.createClass({displayName: "MenuPage",
	mixins: [Routing.RouterNavigateMixin],
	getInitialState: function(): any {
		return {
			menu_json: null,
			busy: false,
			error: null
		};
	},
	componentWillMount: function() {
		document.title = "Pathways | Menu";
		this.setState({busy: true});
		AdminAPI.pathwayMenu(function(success, data, error) {
			if (this.isMounted()) {
				if (success) {
					var menuJSON = JSON.stringify(data, null, 4)
					var parsedMenu = null;
					var error = null;
					try {
						parsedMenu = JSON.parse(menuJSON);
					} catch(ex) {
						error = "Invalid JSON: " + ex.message;
					}
					this.setState({
						busy: false,
						error: null,
						menu_json: menuJSON,
						menu: parsedMenu
					});
				} else {
					this.setState({busy: false, error: error});
				}
			}
		}.bind(this));
	},
	onChange: function(e: any) {
		e.preventDefault();
		var error = null;
		var parsedMenu = null;
		try {
			parsedMenu = JSON.parse(e.target.value)
		} catch(ex) {
			error = "Invalid JSON: " + ex.message;
		}
		this.setState({
			error: error,
			menu_json: e.target.value,
			menu: parsedMenu
		});
	},
	onSubmit: function(e: any) {
		e.preventDefault();
		if (!Perms.has(Perms.PathwaysEdit)) {
			return;
		}
		try {
			var menu = JSON.parse(this.state.menu_json);
		} catch(ex) {
			this.setState({error: "Invalid JSON: " + ex.message});
			return;
		}
		this.setState({busy: true});
		AdminAPI.updatePathwayMenu(menu, function(success, data, error) {
			if (this.isMounted()) {
				if (success) {
					this.setState({
						busy: false,
						error: null,
						menu_json: JSON.stringify(data, null, 4)
					});
				} else {
					this.setState({busy: false, error: error.message});
				}
			}
		}.bind(this));
	},
	render: function(): any {
		var menuEmptyStateDivStyle = {
			position: 'absolute',
			top: '50'
		};
		var pathwaysMenuPreviewContainerStyle = {
			backgroundColor: '#0BA5C5',
			width: '320px',
			height: '568px'
		};
		return (
			<div>
				<div className="row">
					<div className="col-sm-12 col-md-12 col-lg-8">
						<h2>Pathways Menu</h2>
						{this.state.menu_json ?
							<form role="form" onSubmit={this.onSubmit} method="PUT">
								<div>
									{Perms.has(Perms.PathwaysEdit) ?
										<Forms.TextArea name="json" required label="JSON" value={this.state.menu_json} rows="20" onChange={this.onChange} tabs={true} />
									: <pre>{this.state.menu_json}</pre>}
								</div>
								<div className="text-right">
									{this.state.error ? <Utils.Alert type="danger">{this.state.error}</Utils.Alert> : null}
									{this.state.busy ? <Utils.LoadingAnimation /> : null}
									{Perms.has(Perms.PathwaysEdit) ?
										<button type="submit" className="btn btn-primary">Save</button>
									:null}
								</div>
							</form>
						:
							<div>
								{this.state.error ? <Utils.Alert type="danger">{this.state.error}</Utils.Alert> : null}
								{this.state.busy ? <Utils.LoadingAnimation /> : null}
							</div>
						}
					</div>
					<div className="col-sm-12 col-md-12 col-lg-4">
						<h3>Preview</h3>
						<div style={pathwaysMenuPreviewContainerStyle} key="pathways_menu_preview_container">
							<ReactCSSTransitionGroup transitionName="animatable_preview_menu" key="animatable_preview_menu_container">
							{this.state.menu ?
								<PreviewMenuContainer topLevelMenu={this.state.menu} key="preview_menu_container" />
							:
								<div className="pathways_menu_preview_title" style={menuEmptyStateDivStyle}>
									Not Available
								</div>
							}
							</ReactCSSTransitionGroup>
						</div>
					</div>
				</div>
				<div className="row">
					<div className="col-sm-6">
						<h3>Submenu Item</h3>
						<p>
							The structure of a pathway submenu item is:
							<pre>
								{JSON.stringify({
									"title": "",
									"type": "menu",
									"menu": {
										"title": "",
										"items": []
									}
								}, null, 4)}
							</pre>
						</p>
					</div>
					<div className="col-sm-6">
						<h3>Pathway Item</h3>
						<p>
							The structure of a pathway menu item is:
							<pre>
								{JSON.stringify({
									"title": "",
									"type": "pathway",
									"conditionals": [],
									"pathway_tag": ""
								}, null, 4)}
							</pre>
						</p>
					</div>
					<div className="col-sm-16">
						<AvailablePathwaysList />
					</div>
					<div className="col-sm-12">
						<h3>Conditionals</h3>
						<p>
							Each menu item can include a conditions that must all match for that menu item to be shown.
							Supported conditional operators are <code> ==</code>,
							<code>&lt;</code>, <code>&gt;</code>, and <code>in</code>. To make the condition a negative
							then set <code>"not": true</code>, for example for not equals it would be
							<code>{"{"}"op": "==", "key": "gender", "value": "male", "not": true{"}"}</code>
						</p>
						<p>
							Examples (only show Acne menu item for females who are 18 or older in California, Florida, or New York):
							<pre>
								{JSON.stringify({
									"title": "Acne",
									"type": "pathway",
									"conditionals": [
										{"op": "==", "key": "gender", "value": "female"},
										{"op": "in", "key": "state", "value": ["CA", "FL", "NY"]},
										{"op": "<", "key": "age", "value": 18, "not": true}
									],
									"pathway": {
										"tag": "health_condition_acne",
									}
								}, null, 4)}
							</pre>
						</p>
					</div>
				</div>
			</div>
		);
	}
});

var ListPage = React.createClass({displayName: "ListPage",
	mixins: [Routing.RouterNavigateMixin],
	getInitialState: function(): any {
		return {
			pathways: [],
			busy: false,
			error: null
		};
	},
	componentWillMount: function() {
		document.title = "Pathways";
		this.setState({busy: true});
		AdminAPI.pathways(false, function(success, data, error) {
			if (this.isMounted()) {
				if (success) {
					this.setState({
						busy: false,
						error: null,
						pathways: data.pathways
					});
				} else {
					this.setState({
						busy: false,
						error: error.message,
						pathways: []
					});
				}
			}
		}.bind(this));
	},
	onAddPathway: function() {
		// Reload the pathways list
		this.componentWillMount();
	},
	onRename: function(p, e) {
		e.preventDefault();
		var newName = window.prompt("New name", p.name);
		if (newName) {
			p.name = newName;
			this.setState({busy: true, pathways: this.state.pathways});
			AdminAPI.updatePathway(p.id, {name: newName}, function(success, data, error) {
				if (this.isMounted()) {
					if (success) {
						this.setState({busy: false, error: null});
					} else {
						this.setState({busy: false, error: error.message});
					}
				}
			}.bind(this));
		}
	},
	render: function(): any {
		return (
			<div className="container">
				{Perms.has(Perms.PathwaysEdit) ? <AddPathwayModal onSuccess={this.onAddPathway} /> : null}
				<div className="row">
					<div className="col-sm-12 col-md-12 table-responsive">
						<div>
							{Perms.has(Perms.PathwaysEdit) ? <div className="pull-right"><button className="btn btn-default" data-toggle="modal" data-target="#add-pathway-modal">+</button></div> : null}
							<h2>Pathways</h2>
						</div>
						{this.state.error ? <Utils.Alert type="danger">{this.state.error}</Utils.Alert> : null}
						{this.state.busy ? <Utils.LoadingAnimation /> : null}
						<table className="table">
							<thead>
								<tr>
									<th>Name</th>
									<th>Tag</th>
									<th>Branch of Medicine</th>
									<th>Status</th>
								</tr>
							</thead>
							<tbody>
							{this.state.pathways.map(function(p) {
								return (
									<tr key={p.tag}>
										<td>
											<a href={"pathways/details/"+p.id} onClick={this.onNavigate}>{p.name}</a>
											{Perms.has(Perms.PathwaysEdit) ?
												<small> [<a href="#" onClick={this.onRename.bind(this, p)}>rename</a>]</small>
											: null}
										</td>
										<td>{p.tag}</td>
										<td>{p.medicine_branch}</td>
										<td>{p.status}</td>
									</tr>
								);
							}.bind(this))}
							</tbody>
						</table>
					</div>
				</div>
			</div>
		);
	}
});

var DetailsPage = React.createClass({displayName: "DetailsPage",
	mixins: [Routing.RouterNavigateMixin],
	getInitialState: function(): any {
		return {
			pathway: null,
			details_json: null,
			busy: false,
			error: null
		};
	},
	componentWillMount: function() {
		document.title = "Pathway Details";
		this.setState({details_busy: true});
		AdminAPI.pathway(this.props.pathwayID, function(success, data, error) {
			if (this.isMounted()) {
				if (success) {
					document.title = data.pathway.name + " Pathway Details";
					this.setState({
						details_busy: false,
						details_error: null,
						pathway: data.pathway,
						details_json: JSON.stringify(data.pathway.details, null, 4)
					});
				} else {
					this.setState({
						details_busy: false,
						error: error.message
					});
				}
			}
			this.setState({stp_busy: true});
			AdminAPI.sampleTreatmentPlan(this.state.pathway.tag, function(success, data, error) {
				if (this.isMounted()) {
					if (success) {
						this.setState({
							stp_busy: false,
							stp_error: null,
							stp_json: JSON.stringify(data, null, 4)
						});
					} else {
						this.setState({
							stp_busy: false,
							stp_error: error.message
						});
					}
				}
			}.bind(this));
		}.bind(this));
	},
	onDetailsChange: function(e) {
		e.preventDefault();
		var error = null;
		try {
			JSON.parse(e.target.value)
		} catch(ex) {
			error = "Invalid JSON: " + ex.message;
		}
		this.setState({
			details_error: error,
			details_json: e.target.value
		});
	},
	onSTPChange: function(e) {
		e.preventDefault();
		var error = null;
		try {
			JSON.parse(e.target.value)
		} catch(ex) {
			error = "Invalid JSON: " + ex.message;
		}
		this.setState({
			stp_error: error,
			stp_success_text: null,
			stp_json: e.target.value
		});
	},
	onDetailsSubmit: function(e) {
		e.preventDefault();
		if (!Perms.has(Perms.PathwaysEdit)) {
			return;
		}
		try {
			var details = JSON.parse(this.state.details_json);
		} catch(ex) {
			this.setState({details_error: "Invalid JSON: " + ex.message});
			return;
		}
		this.setState({details_busy: true});
		AdminAPI.updatePathway(this.props.pathwayID, {details: details}, function(success, data, error) {
			if (this.isMounted()) {
				if (success) {
					this.setState({
						details_busy: false,
						details_error: null,
						pathway: data.pathway,
						details_json: JSON.stringify(data.details, null, 4)
					});
				} else {
					this.setState({details_busy: false, details_error: error.message});
				}
			}
		}.bind(this));
	},
	onSTPSubmit: function(e) {
		e.preventDefault();
		if (!Perms.has(Perms.STPEdit)) {
			return;
		}
		try {
			var stp = JSON.parse(this.state.stp_json);
		} catch(ex) {
			this.setState({stp_error: "Invalid JSON: " + ex.message});
			return;
		}
		this.setState({stp_busy: true});
		AdminAPI.updateSampleTreatmentPlan(this.state.pathway.tag, stp, function(success, data, error) {
			if (this.isMounted()) {
				if (success) {
					this.setState({
						stp_busy: false,
						stp_error: null,
						stp_success_text: "Sample Treatment Plan Sucessfully Updated"
					});
				} else {
					this.setState({
						stp_busy: false,
						stp_error: error.message,
						stp_success_text: null
					});
				}
			}
		}.bind(this));
	},
	render: function(): any {
		return (
			<div className="container">
				<div className="row">
					<div className="col-sm-12 col-md-12">
						{this.state.pathway ?
							<div>
								<h2>{this.state.pathway.name} Pathway</h2>
								<form role="form" onSubmit={this.onDetailsSubmit} method="PUT">
									<div>
										{Perms.has(Perms.PathwaysEdit) ?
											<Forms.TextArea name="json" required label="Pathway Details JSON" value={this.state.details_json} rows="20" onChange={this.onDetailsChange} tabs={true} />
										:
											<pre>{this.state.details_json}</pre>
										}
									</div>
									<div className="text-right">
										{this.state.details_error ? <Utils.Alert type="danger">{this.state.details_error}</Utils.Alert> : null}
										{this.state.details_busy ? <Utils.LoadingAnimation /> : null}
										{Perms.has(Perms.STPEdit) ?
											<button type="submit" className="btn btn-primary">Save</button>
										:null}
									</div>
								</form>
								<form role="form" onSubmit={this.onSTPSubmit} method="PUT">
									<div>
										{Perms.has(Perms.STPEdit) ?
											<Forms.TextArea name="stp_json" required label="Sample Treatment Plan JSON" value={this.state.stp_json} rows="20" onChange={this.onSTPChange} tabs={true} />
										:
											<pre>{this.state.stp_json}</pre>
										}
									</div>
									<div className="text-right">
										{this.state.stp_error ? <Utils.Alert type="danger">{this.state.stp_error}</Utils.Alert> : null}
										{this.state.stp_success_text ? <Utils.Alert type="success">{this.state.stp_success_text}</Utils.Alert> : null}
										{this.state.stp_busy ? <Utils.LoadingAnimation /> : null}
										{Perms.has(Perms.STPEdit) ?
											<button type="submit" className="btn btn-primary">Save</button>
										:null}
									</div>
								</form>
							</div>
						:
							<div>
								<h2>Pathway</h2>
								{this.state.error ? <Utils.Alert type="danger">{this.state.error}</Utils.Alert> : null}
								{this.state.busy ? <Utils.LoadingAnimation /> : null}
							</div>
						}
					</div>
				</div>
			</div>
		);
	}
});

var AvailablePathwaysList = React.createClass({displayName: "AvailablePathwaysList",
	mixins: [Routing.RouterNavigateMixin],
	getInitialState: function(): any {
		return {
			pathways: [],
			busy: false,
			error: null
		};
	},
	componentWillMount: function() {
		this.setState({busy: true});
		AdminAPI.pathways(true, function(success, data, error) {
			if (this.isMounted()) {
				if (success) {
					this.setState({
						busy: false,
						error: null,
						pathways: data.pathways
					});
				} else {
					this.setState({
						busy: false,
						error: error.message
					});
				}
			}
		}.bind(this));
	},
	render: function(): any {
		return (
			<div className="pathway-list">
				<h3>Available Pathways</h3>
				{this.state.error ? <Utils.Alert type="danger">{this.state.error}</Utils.Alert> : null}
				{this.state.busy ? <Utils.LoadingAnimation /> : null}
				<ul>
				{this.state.pathways.map(function(p) {
					return (
						<li key={p.tag}>{p.name} [{p.tag}]</li>
					);
				})}
				</ul>
			</div>
		);
	}
});

var AvailableIntakeTemplatesList = React.createClass({displayName: "AvailableIntakeTemplatesList",
	mixins: [Routing.RouterNavigateMixin],
	getInitialState: function(): any {
		return {
			versions: [],
			busy: false,
			error: null
		};
	},
	onClick: function(e: any) {
		e.preventDefault()
		this.props.onClick(e.target.text)
	},
	render: function(): any {
		return (
			<div className="intake-version-list">
				<h3>Available Intake Templates</h3>
				{this.state.error ? <Utils.Alert type="danger">{this.state.error}</Utils.Alert> : null}
				{this.state.busy ? <Utils.LoadingAnimation /> : null}
				<ul>
				{
					this.props.intake_versions.map(function(v) {
					return (
						<li key={v}><a text={v} onClick={this.onClick} href="#">{v}</a></li>
					);
				}.bind(this))}
				</ul>
			</div>
		);
	}
});

var AvailableReviewTemplatesList = React.createClass({displayName: "AvailableReviewTemplatesList",
	mixins: [Routing.RouterNavigateMixin],
	getInitialState: function(): any {
		return {
			versions: [],
			busy: false,
			error: null
		};
	},
	onClick: function(e: any) {
		e.preventDefault()
		this.props.onClick(e.target.text)
	},
	render: function(): any {
		return (
			<div className="intake-version-list">
				<h3>Available Review Templates</h3>
				{this.state.error ? <Utils.Alert type="danger">{this.state.error}</Utils.Alert> : null}
				{this.state.busy ? <Utils.LoadingAnimation /> : null}
				<ul>
				{
					this.props.review_versions.map(function(v) {
					return (
						<li key={v}><a text={v} onClick={this.onClick} href="#">{v}</a></li>
					);
				}.bind(this))}
				</ul>
			</div>
		);
	}
});

var AvailablePathwaysSelect = React.createClass({displayName: "AvailablePathwaysSelect",
	mixins: [Routing.RouterNavigateMixin],
	getInitialState: function(): any {
		return {
			pathway_tags: [],
			pathway_names: {},
			busy: false,
			error: null
		};
	},
	componentWillMount: function() {
		this.setState({busy: true});
		AdminAPI.pathways(true, function(success, data, error) {
			if (this.isMounted()) {
				if (success) {
					var pathway_tags = []
					var pathway_names = {}
					for (var i = 0; i < data.pathways.length; i++) {
						pathway_tags.push({name: data.pathways[i].tag, value: data.pathways[i].tag})
						pathway_names[data.pathways[i].tag] = data.pathways[i].name
					}
					this.setState({
						busy: false,
						error: null,
						pathway_tags: pathway_tags,
						pathway_names: pathway_names,
						selected_value: pathway_tags.length == 0  ? "" : pathway_tags[0].value
					});
				} else {
					this.setState({
						busy: false,
						error: error.message
					});
				}
			}
		}.bind(this));
	},
	onChange: function(e) {
		this.props.onChange(e, e.target.value, this.state.pathway_names[e.target.value])
		this.setState({
						selected_value: e.target.value
					});
	},
	render: function(): any {
		return (
			<div className="pathways-select">
				<form>
					<Forms.FormSelect onChange={this.onChange} value={this.state.selected_value} opts={this.state.pathway_tags} />
				</form>
			</div>
		);
	}
});

var AddPathwayModal = React.createClass({displayName: "AddPathwayModal",
	getInitialState: function(): any {
		return this.stateForProps(this.props);
	},
	stateForProps: function(props) {
		return {
			error: "",
			busy: false,
			name: "",
			tag: "",
			medicineBranch: ""
		}
	},
	componentWillReceiveProps: function(nextProps) {
		this.setState(this.stateForProps(nextProps));
	},
	onChangeName: function(e) {
		e.preventDefault();
		this.setState({error: "", name: e.target.value});
	},
	onChangeTag: function(e) {
		e.preventDefault();
		this.setState({error: "", tag: e.target.value});
	},
	onChangeMedicineBranch: function(e) {
		e.preventDefault();
		this.setState({error: "", medicineBranch: e.target.value});
	},
	onAdd: function(e) {
		if (!this.state.name) {
			this.setState({error: "name is required"});
			return true;
		}
		if (!this.state.tag) {
			this.setState({error: "tag is required"});
			return true;
		}
		if (!this.state.medicineBranch) {
			this.setState({error: "medicine branch is required"});
			return true;
		}
		this.setState({busy: true, error: ""});
		var pathway = {
			name: this.state.name,
			tag: this.state.tag,
			medicine_branch: this.state.medicineBranch
		};
		AdminAPI.createPathway(pathway, function(success, data, error) {
			if (this.isMounted()) {
				if (!success) {
					this.setState({busy: false, error: error.message});
					return;
				}
				this.setState({busy: false});
				this.props.onSuccess();
				$("#add-pathway-modal").modal('hide');
			}
		}.bind(this));
		return true;
	},
	render: function(): any {
		return (
			<Modals.ModalForm id="add-pathway-modal" title="Add Pathway"
				cancelButtonTitle="Cancel" submitButtonTitle="Add"
				onSubmit={this.onAdd}>

				{this.state.error ? <Utils.Alert type="danger">{this.state.error}</Utils.Alert> : null}

				<Forms.FormInput label="Name" value={this.state.name} onChange={this.onChangeName} />
				<Forms.FormInput label="Tag" value={this.state.tag} onChange={this.onChangeTag} />
				<Forms.FormInput label="Branch of Medicine" value={this.state.medicineBranch} onChange={this.onChangeMedicineBranch} />
			</Modals.ModalForm>
		);
	}
});

var SAMLPage = React.createClass({displayName: "SAMLPage",
	getInitialState: function(): any {
		return {
			saml: "",
			yaml: "",
			busy: false,
			error: null,
		};
	},
	handleSAMLChange: function(e) {
		e.preventDefault();
		this.setState({saml: e.target.value});
	},
	handleYAMLChange: function(e) {
		e.preventDefault();
		this.setState({yaml: e.target.value});
	},
	handleTransform: function(e) {
		e.preventDefault();
		this.setState({busy: true});
		AdminAPI.transformSAML(this.state.saml, function(success, data, error) {
			if (this.isMounted()) {
				if (!success) {
					this.setState({busy: false, error: error.message});
					return;
				}
				var yaml = "";
				if (data.intake) {
					yaml = jsyaml.safeDump(data.intake);
				}
				this.setState({busy: false, error: data.error, yaml: yaml});
			}
		}.bind(this));
	},
	render: function(): any {
		return (
			<div>
				<Forms.TextArea label="SAML" rows="20" value={this.state.saml} onChange={this.handleSAMLChange} />
				<div className="text-right">
					<button onClick={this.handleTransform}>Transform</button>
					{this.state.error ? <Utils.Alert type="danger">{this.state.error}</Utils.Alert> : null}
					{this.state.busy ? <Utils.LoadingAnimation /> : null}
				</div>
				<Forms.TextArea label="YAML" rows="20" value={this.state.yaml} onChange={this.handleYAMLChange} />
			</div>
		);
	}
});
