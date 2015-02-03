/** @jsx React.DOM */

var AdminAPI = require("./api.js");
var IntakeReview = require("./intake_review.js");
var Forms = require("../forms.js");
var Modals = require("../modals.js");
var Nav = require("../nav.js");
var Perms = require("./permissions.js");
var Routing = require("../routing.js");
var Utils = require("../utils.js");

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
			}
		]],
		getDefaultProps: function() {
			return {}
		},
		list: function() {
			return <ListPage router={this.props.router} />;
		},
		menu: function() {
			return <MenuPage router={this.props.router} />;
		},
		intake_templates: function() {
			return <IntakeTemplatesPage router={this.props.router} />;
		},
		details: function() {
			return <DetailsPage router={this.props.router} pathwayID={this.props.pathwayID} />;
		},
		render: function() {
			return (
				<div>
					<Nav.LeftNav router={this.props.router} items={this.menuItems} currentPage={this.props.page}>
						{this[this.props.page]()}
					</Nav.LeftNav>
				</div>
			);
		}
	})
};

var IntakeTemplatesPage = React.createClass({displayName: "IntakeTemplatesPage",
	mixins: [Routing.RouterNavigateMixin],
	getInitialState: function() {
		return {
			intake_json: JSON.stringify(this.intake_spec, null, 4),
			review_json: "",
			busy: false,
			error: null,
			pathway_tag: null,
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
		AdminAPI.pathways(true, function(success, data, error) {
			tags = []
			for(i in data.pathways) {
				tags.push(data.pathways[i].tag)
			}
			this.setState({
					busy: false,
					pathway_tag: tags[0]
				})
			this.updatePathwayVersions(tags[0])
		}.bind(this));
	},
	onChangeIntake: function(e) {
		e.preventDefault();
		var error = null;
		try {
			JSON.parse(e.target.value)
		} catch(ex) {
			error = "Invalid JSON: " + ex.message;
		}
		this.setState({
			intake_error: error,
			intake_json: e.target.value
		});
	},
	onChangeReview: function(e) {
		e.preventDefault();
		var error = null;
		try {
			JSON.parse(e.target.value)
		} catch(ex) {
			error = "Invalid JSON: " + ex.message;
		}
		this.setState({
			review_error: error,
			review_json: e.target.value
		});
	},
	onPathwayChange: function(e, pathway_tag) {
		this.updatePathwayVersions(pathway_tag)
	},
	updatePathwayVersions: function(pathway_tag) {
		AdminAPI.layoutVersions(function(success, data, error) {
			intake_versions = [];
			review_versions = [];
			newest_intake_version = undefined
			newest_review_version = undefined
			for(pt in data) {
				if(pt == pathway_tag) {
					for(p in data[pt]){
						if(p == "CONDITION_INTAKE") {
							intake_versions = data[pt][p];
							newest_intake_version = data[pt][p][data[pt][p].length-1]
						}
						if(p == "REVIEW") {
							review_versions = data[pt][p];
							newest_review_version = data[pt][p][data[pt][p].length-1]
						}
					}
				}
			}
			intake_json = intake_versions.length == 0 ? JSON.stringify(this.intake_spec, null, 4) : this.state.intake_json 
			this.setState({
				pathway_tag: pathway_tag,
				intake_json: intake_json,
				intake_versions: intake_versions,
				newest_intake_version: newest_intake_version,
				review_versions: review_versions,
				newest_review_version: newest_review_version,
			});
		}.bind(this));
	},
	onIntakeVersionSelection: function(version) {
		version = version.split(".")
		AdminAPI.template(this.state.pathway_tag, "CONDITION_INTAKE", version[0], version[1], version[2], function(success, data, error) {
			try {
				data = IntakeReview.expandTemplate(data)
				this.setState({
					intake_json: JSON.stringify(data, null, 4),
				});
			} catch (e) {
				this.setState({
					intake_error: e.message,
				});
			}
		}.bind(this));
	},
	onReviewVersionSelection: function(version) {
		version = version.split(".")
		AdminAPI.template(this.state.pathway_tag, "REVIEW", version[0], version[1], version[2], function(success, data, error) {
			this.setState({
				review_json: JSON.stringify(data, null, 4),
			});
		}.bind(this));
	},
	generateReview: function(e) {
		e.preventDefault();
		try {
			review = IntakeReview.generateReview(JSON.parse(this.state.intake_json), this.state.pathway_tag)
			this.setState({
				review_json: JSON.stringify(review, null, 4),
			});
		} catch (e) {
			this.setState({
				intake_error: e.message,
			});
		}
	},
	submitLayout: function(e) {
		e.preventDefault();
		this.setState({busy: true});
		intake = JSON.parse(this.state.intake_json)
		review = JSON.parse(this.state.review_json)
		if(this.state.newest_intake_version != undefined) {
			intake_v = this.state.newest_intake_version.split(".")
		} else {
			intake_v = ["1","-1","0"]
		}
		if(this.state.newest_review_version != undefined) {
			review_v = this.state.newest_review_version.split(".")
		} else {
			review_v = ["1","-1","0"]
		}
		intake.version = intake_v[0] + "." + (parseInt(intake_v[1]) + 1) + "." + intake_v[2]
		review.version = review_v[0] + "." + (parseInt(review_v[1]) + 1) + "." + review_v[2]
		try {
			IntakeReview.submitLayout(intake, review, this.state.pathway_tag)
			this.setState({
				success_text: "Intake " + intake.version + " and Review " + review.version + " created for Pathway " + this.state.pathway_tag,
				review_error: null,
				busy: false,
			})
			this.updatePathwayVersions(this.state.pathway_tag)
		} catch (e) {
			this.setState({
				success_text: null,
				review_error: e.message,
				busy: false,
			})
		}
	},
	render: function() {
		return (
			<div>
				<div className="row">
					<div className="col-sm-12 col-md-12 col-lg-9">
						<h2>Pathway Templates</h2>
						<AvailablePathwaysSelect onChange={this.onPathwayChange} />
						{this.state.intake_json ?
							<form role="form" onSubmit={this.onSubmit} method="PUT">
								<div>
									{Perms.has(Perms.LayoutEdit) ?
										<Forms.TextArea name="json" required label="Intake Template" value={this.state.intake_json} rows="20" onChange={this.onChangeIntake} tabs={true} />
									: <pre>{this.state.intake_json}</pre>}
									{this.state.intake_error ? <Utils.Alert type="danger">{this.state.intake_error}</Utils.Alert> : null}
								</div>
								<button className="btn" href="#" onClick={this.generateReview}>Generate Review</button>
								<div>
									{Perms.has(Perms.LayoutEdit) ?
										<Forms.TextArea name="json" required label="Review Template" value={this.state.review_json} rows="20" onChange={this.onChangeReview} tabs={true} />
									: <pre>{this.state.review_json}</pre>}
									{this.state.review_error ? <Utils.Alert type="danger">{this.state.review_error}</Utils.Alert> : null}
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

var MenuPage = React.createClass({displayName: "MenuPage",
	mixins: [Routing.RouterNavigateMixin],
	getInitialState: function() {
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
	onChange: function(e) {
		e.preventDefault();
		var error = null;
		try {
			JSON.parse(e.target.value)
		} catch(ex) {
			error = "Invalid JSON: " + ex.message;
		}
		this.setState({
			error: error,
			menu_json: e.target.value
		});
	},
	onSubmit: function(e) {
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
	render: function() {
		return (
			<div>
				<div className="row">
					<div className="col-sm-12 col-md-12 col-lg-9">
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
					<div className="col-sm-12 col-md-12 col-lg-3">
						<AvailablePathwaysList />
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
	getInitialState: function() {
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
	render: function() {
		return (
			<div className="container">
				{Perms.has(Perms.PathwaysEdit) ? <AddPathwayModal onSuccess={this.onAddPathway} /> : null}
				<div className="row">
					<div className="col-sm-12 col-md-12">
						{Perms.has(Perms.PathwaysEdit) ? <div className="pull-right"><button className="btn btn-default" data-toggle="modal" data-target="#add-pathway-modal">+</button></div> : null}
						<h2>Pathways</h2>
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
										<td><a href={"pathways/details/"+p.id} onClick={this.onNavigate}>{p.name}</a></td>
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
	getInitialState: function() {
		return {
			pathway: null,
			details_json: null,
			busy: false,
			error: null
		};
	},
	componentWillMount: function() {
		document.title = "Pathway Details";
		this.setState({busy: true});
		AdminAPI.pathway(this.props.pathwayID, function(success, data, error) {
			if (this.isMounted()) {
				if (success) {
					document.title = data.pathway.name + " Pathway Details";
					this.setState({
						busy: false,
						error: null,
						pathway: data.pathway,
						details_json: JSON.stringify(data.pathway.details, null, 4)
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
		e.preventDefault();
		var error = null;
		try {
			JSON.parse(e.target.value)
		} catch(ex) {
			error = "Invalid JSON: " + ex.message;
		}
		this.setState({
			error: error,
			details_json: e.target.value
		});
	},
	onSubmit: function(e) {
		e.preventDefault();
		if (!Perms.has(Perms.PathwaysEdit)) {
			return;
		}
		try {
			var details = JSON.parse(this.state.details_json);
		} catch(ex) {
			this.setState({error: "Invalid JSON: " + ex.message});
			return;
		}
		this.setState({busy: true});
		AdminAPI.updatePathway(this.props.pathwayID, details, function(success, data, error) {
			if (this.isMounted()) {
				if (success) {
					this.setState({
						busy: false,
						error: null,
						pathway: data.pathway,
						details_json: JSON.stringify(data.details, null, 4)
					});
				} else {
					this.setState({busy: false, error: error.message});
				}
			}
		}.bind(this));
	},
	render: function() {
		return (
			<div className="container">
				<div className="row">
					<div className="col-sm-12 col-md-12">
						{this.state.pathway ?
							<div>
								<h2>{this.state.pathway.name} Pathway</h2>
								<form role="form" onSubmit={this.onSubmit} method="PUT">
									<div>
										{Perms.has(Perms.PathwaysEdit) ?
											<Forms.TextArea name="json" required label="JSON" value={this.state.details_json} rows="20" onChange={this.onChange} tabs={true} />
										:
											<pre>{this.state.details_json}</pre>
										}
									</div>
									<div className="text-right">
										{this.state.error ? <Utils.Alert type="danger">{this.state.error}</Utils.Alert> : null}
										{this.state.busy ? <Utils.LoadingAnimation /> : null}
										{Perms.has(Perms.PathwaysEdit) ?
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
	getInitialState: function() {
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
	render: function() {
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
	getInitialState: function() {
		return {
			versions: [],
			busy: false,
			error: null
		};
	},
	componentWillMount: function() {
		this.setState({busy: false});
		if (this.isMounted()) {
			if (success) {
				this.setState({
					busy: false,
					error: null,
				});
			} else {
				this.setState({
					busy: false,
					error: error.message
				});
			}
		}
	},
	onClick: function(e) {
		e.preventDefault()
		this.props.onClick(e.target.text)
	},
	render: function() {
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
	getInitialState: function() {
		return {
			versions: [],
			busy: false,
			error: null
		};
	},
	componentWillMount: function() {
		this.setState({busy: false});
		if (this.isMounted()) {
			if (success) {
				this.setState({
					busy: false,
					error: null,
				});
			} else {
				this.setState({
					busy: false,
					error: error.message
				});
			}
		}
	},
	onClick: function(e) {
		e.preventDefault()
		this.props.onClick(e.target.text)
	},
	render: function() {
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
	getInitialState: function() {
		return {
			pathway_tags: [],
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
					for(i in data.pathways){
						pathway_tags.push({name: data.pathways[i].tag, value: data.pathways[i].tag})
					}
					this.setState({
						busy: false,
						error: null,
						pathway_tags: pathway_tags,
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
		this.props.onChange(e, e.target.value)
		this.setState({
						selected_value: e.target.value
					});
	},
	render: function() {
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
	getInitialState: function() {
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
	render: function() {
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
