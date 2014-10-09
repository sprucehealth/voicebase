/** @jsx React.DOM */

var AdminAPI = require("./api.js");
var Forms = require("../forms.js");
var Modals = require("../modals.js");
var Nav = require("../nav.js");
var Perms = require("./permissions.js");
var Routing = require("../routing.js");
var Utils = require("../utils.js");

module.exports = {
	EmailAdmin: React.createClass({displayName: "EmailAdmin",
		mixins: [Routing.RouterNavigateMixin],
		getInitialState: function() {
			return {
				types: null,
				senders: null,
				templates: null
			};
		},
		menuItems: function() {
			var menuItems = [];
			var typeMenu = [
				{id: "types-label", name: "Types", url: "/admin/email", heading: true}
			];
			if (this.state.types == null) {
				typeMenu.push({
					id: "loading-types",
					name: (<Utils.LoadingAnimation />),
					url: "#",
					onClick: Utils.swallowEvent
				});
			} else {
				for(var key in this.state.types) {
					var type = this.state.types[key];
					var url = "/admin/email/" + key;
					if (this.state.templates != null) {
						var ts = this.state.templates[key];
						if (ts != null && ts.length != 0) {
							url = url + "/" + ts[0].id;
						}
					}
					typeMenu.push({
						id: key,
						name: type.name,
						url: url,
						active: key == this.props.typeKey
					});
				}
			}
			menuItems.push(typeMenu);

			if (this.state.templates != null) {
				var templates = this.state.templates[this.props.typeKey] || [];
				var templatesMenu = [
					{id: "templates-label", name: "Templates", url: "/admin/email/" + this.props.typeKey, heading: true}
				];
				for(var i = 0; i < templates.length; i++) {
					var tmpl = templates[i];
					templatesMenu.push({
						id: tmpl.id,
						name: (
							<span>
								<span className={"glyphicon glyphicon-" + (tmpl.active?"ok":"remove")} />
								&nbsp;{tmpl.name}
							</span>
						),
						url: "/admin/email/" + tmpl.type + "/" + tmpl.id,
						active: tmpl.id == this.props.templateID
					});
				}
				if (Perms.has(Perms.EmailEdit)) {
					templatesMenu.push({
						id: "new-template",
						url: "/admin/email/" + this.props.typeKey + "/_new",
						name: (<span><span className="glyphicon glyphicon-plus" /> New Template</span>),
						active: this.props.templateID == "_new"
					});
				}
				menuItems.push(templatesMenu);
			}
			return menuItems;
		},
		componentWillReceiveProps: function(nextProps) {
			if (nextProps.typeKey == null) {
				for(var key in this.state.types) {
					setTimeout(function() { this.navigateToSomething(nextProps, key); }.bind(this), 50);
					break;
				}
			} else if (nextProps.templateID == null) {
				setTimeout(function() { this.navigateToSomething(nextProps); }.bind(this), 50);
			}
		},
		componentWillMount: function() {
			document.title = "Email | Spruce Admin";

			this.loadTypes();
			this.loadSenders();
			this.loadTemplates("");
		},
		loadSenders: function() {
			AdminAPI.listEmailSenders(function(success, data, error) {
				if (this.isMounted()) {
					if (!success) {
						// TODO
						alert("Failed to get email senders: " + error.message);
						return;
					}
				}
				this.setState({senders: data || []});
			}.bind(this));
		},
		loadTypes: function() {
			AdminAPI.listEmailTypes(function(success, data, error) {
				if (this.isMounted()) {
					if (!success) {
						// TODO
						alert("Failed to get email types: " + error.message);
						return;
					}
				}
				var types = data || [];
				this.setState({types: types});
				if (this.props.typeKey == null && types.length != 0) {
					for(var key in types) {
						this.navigate("/email/" + key);
						break;
					}
				}
			}.bind(this));
		},
		navigateToSomething: function(props, typeKey) {
			var t = null;
			typeKey = typeKey || props.typeKey;
			if (typeKey != null) {
				t = this.state.templates[typeKey][0];
			} else {
				for(var key in this.state.templates) {
					var ts = this.state.templates[key];
					if (ts != null && ts.length != 0) {
						t = ts[0];
						break;
					}
				}
			}
			if (t != null) {
				this.navigate("/email/" + t.type + "/" + t.id);
			}
		},
		loadTemplates: function(typeKey) {
			AdminAPI.listEmailTemplates(typeKey, function(success, data, error) {
				if (this.isMounted()) {
					if (!success) {
						// TODO
						alert("Failed to get templates list: " + error.message);
						return;
					}
					data = data || [];
					var templates = {};
					for(var i = 0; i < data.length; i++) {
						var t = data[i];
						var ts = templates[t.type];
						if (!ts) {
							ts = [];
							templates[t.type] = ts;
						}
						ts.push(t);
					}
					this.setState({
						templates: templates
					});
					if (this.props.templateID == null) {
						this.navigateToSomething(this.props);
					}
				}
			}.bind(this));
		},
		onSaveTemplate: function(templateID) {
			this.loadTemplates("");
			this.navigate("/email/" + this.props.typeKey + "/" + templateID)
		},
		render: function() {
			var currentPage = this.props.page;
			// if (currentPage == "reports") {
			// 	currentPage = "report-" + this.props.reportID;
			// }

			var content = "";
			if (this.state.types == null) {
				content = <Utils.LoadingAnimation />;
			} else if (this.props.templateID == "_new") {
				content = <EmailEditTemplate router={this.props.router} senders={this.state.senders} type={this.state.types[this.props.typeKey]} onSuccess={this.onSaveTemplate} />;
			} else if (this.props.templateID != null) {
				if (this.state.templates == null) {
					content = <Utils.LoadingAnimation />;
				} else {
					var template = null;
					var ts = this.state.templates[this.props.typeKey];
					for(var i = 0; i < ts.length; i++) {
						var t = ts[i];
						if (t.id == this.props.templateID) {
							template = t;
							break;
						}
					}
					if (template == null) {
						content = "Template Not Found"
					} else {
						if (this.props.edit) {
							content = <EmailEditTemplate router={this.props.router} senders={this.state.senders} type={this.state.types[this.props.typeKey]} onSuccess={this.onSaveTemplate} template={template} />;
						} else {
							content = <EmailTemplate router={this.props.router} senders={this.state.senders} type={this.state.types[this.props.typeKey]} template={template} />;
						}
					}
				}
			}

			return (
				<div>
					<Nav.LeftNav router={this.props.router} items={this.menuItems()} currentPage={currentPage}>
						{content}
					</Nav.LeftNav>
				</div>
			);
		}
	})
};

var EmailTemplate = React.createClass({displayName: "EmailTemplate",
	mixins: [Routing.RouterNavigateMixin],
	onEdit: function(e) {
		e.preventDefault();
		this.navigate("/email/" + this.props.template.type + "/" + this.props.template.id + "/edit");
		return false;
	},
	render: function() {
		var sender = "";
		if (this.props.senders != null) {
			for(var i = 0; i < this.props.senders.length; i++) {
				var s = this.props.senders[i];
				if (s.id == this.props.template.sender_id) {
					sender = Utils.formatEmailAddress(s.name, s.email);
					break;
				}
			}
		}
		return (
			<div>
				<EmailTestModal type={this.props.type} template={this.props.template} />

				{Perms.has(Perms.EmailEdit) ?
					<div className="pull-right">
						<button className="btn btn-default" data-toggle="modal" data-target="#email-test-modal">Test</button>
						&nbsp;<button className="btn btn-default" onClick={this.onEdit}>Edit</button>
					</div>
					: null}

				<h1>{this.props.template.name} <small>[{this.props.template.active?"Active":"Inactive"}]</small></h1>

				<br />

				<div>
					<strong>Sender:</strong> {sender}
				</div>

				<br />

				<div>
					<strong>Subject</strong> {this.props.template.subject_template}
				</div>

				<br />

				<div>
					<div><strong>HTML Body</strong></div>
					<pre>{this.props.template.body_html_template}</pre>
				</div>

				<br />

				<div>
					<div><strong>Text Body</strong></div>
					<pre>{this.props.template.body_text_template}</pre>
				</div>

				<br />

				<div>
					<div><strong>Example Context</strong></div>
					<pre>
						{JSON.stringify(this.props.type.test_context, null, 4)}
					</pre>
				</div>
			</div>
		);
	}
});

var EmailTestModal = React.createClass({displayName: "EmailTestModal",
	getInitialState: function() {
		return this.stateForProps(this.props);
	},
	stateForProps: function(props) {
		return {
			error: "",
			busy: false,
			to: Spruce.Account.email,
			context: JSON.stringify(props.type.test_context, null, 4)
		}
	},
	componentWillReceiveProps: function(nextProps) {
		if (nextProps.type.key != this.props.type.key) {
			this.setState(this.stateForProps(nextProps));
		}
	},
	onSendTest: function(e) {
		var ctx;
		try {
			ctx = JSON.parse(this.state.context)
		} catch(e) {
			this.setState({busy: false, error: "Context is not valid JSON: " + e.toString()});
			return true;
		}

		this.setState({busy: true, error: ""});

		AdminAPI.testEmailTemplate(this.props.template.id, this.state.to, ctx, function(success, data, error) {
			if (this.isMounted()) {
				if (!success) {
					this.setState({busy: false, error: error.message});
					return;
				}
				this.setState({busy: false});
				$("#email-test-modal").modal('hide');
			}
		}.bind(this));
		return true;
	},
	onChangeTo: function(e) {
		this.setState({error: "", to: e.target.value});
	},
	onChangeContext: function(e) {
		this.setState({error: "", context: e.target.value});
	},
	render: function() {
		return (
			<Modals.ModalForm id="email-test-modal" title={<span>Send Test Email {this.state.busy ? <Utils.LoadingAnimation /> : null}</span>}
				cancelButtonTitle="Cancel" submitButtonTitle="Send"
				onSubmit={this.onSendTest}>

				{this.state.error ? <Utils.Alert type="danger">{this.state.error}</Utils.Alert> : null}

				<Forms.FormInput label="To" value={this.state.to} onChange={this.onChangeTo} />
				<Forms.TextArea label="Context" value={this.state.context} onChange={this.onChangeContext} />
			</Modals.ModalForm>
		);
	}
});

var EmailEditTemplate = React.createClass({displayName: "EmailEditTemplate",
	mixins: [Routing.RouterNavigateMixin],
	getInitialState: function() {
		if (this.props.template) {
			return {template: jQuery.extend({}, this.props.template)};
		}
		var sender_id = null;
		if (this.props.senders != null && this.props.senders.length != 0) {
			sender_id = this.props.senders[0].id;
		}
		return {
			template: {
				id: null,
				sender_id: sender_id,
				type: this.props.type.key,
				name: "",
				subject_template: "",
				body_text_template: "",
				body_html_template: "",
				active: false
			}
		};
	},
	componentWillReceiveProps: function(nextProps) {
		if (this.state.template.sender_id == null && nextProps.senders != null && nextProps.senders.length != 0) {
			var tmpl = this.state.template;
			tmpl.sender_id = nextProps.senders[0].id;
			this.setState({template: tmpl});
		}
	},
	onChange: function(e) {
		e.preventDefault();
		var tmpl = this.state.template;
		var oldValue = tmpl[e.target.name];
		if (typeof oldValue == "boolean") {
			tmpl[e.target.name] = e.target.checked;
		} else if (typeof oldValue == "number") {
			tmpl[e.target.name] = Number(e.target.value);
		} else {
			tmpl[e.target.name] = e.target.value;
		}
		this.setState({template: tmpl});
		return false;
	},
	onSubmit: function(e) {
		e.preventDefault();
		if (this.state.template.id == null) {
			AdminAPI.createEmailTemplate(this.state.template, function(success, data, error) {
				if (this.isMounted()) {
					if (!success) {
						// TODO
						alert("Failed to create template: " + error.message);
						return;
					}
					this.props.onSuccess(data);
				}
			}.bind(this));
		} else {
			AdminAPI.updateEmailTemplate(this.state.template, function(success, data, error) {
				if (this.isMounted()) {
					if (!success) {
						alert("Failed to save template: " + error.message);
						return;
					}
					this.props.onSuccess(this.state.template.id);
				}
			}.bind(this));
		}
		return false;
	},
	senderOptions: function() {
		return (this.props.senders || []).map(function(s) {
			return {name: Utils.formatEmailAddress(s.name, s.email), value: s.id}
		});
	},
	onCancel: function(e) {
		e.preventDefault();
		this.navigate("/email/" + this.state.template.type + "/" + this.state.template.id);
		return false;
	},
	render: function() {
		return (
			<div>
				<h1>{this.state.template.id ? this.state.template.name : "New template for " + this.props.type.name}</h1>

				<form method="POST" onSubmit={this.onSubmit}>
					<div className="pull-right">
						<Forms.Checkbox label="Active" name="active" checked={this.state.template.active} onChange={this.onChange} />
					</div>

					<Forms.FormInput label="Template Name" name="name" required={true} value={this.state.template.name} onChange={this.onChange} />
					<Forms.FormSelect label="Sender" name="sender_id" value={this.state.template.sender_id} onChange={this.onChange} opts={this.senderOptions()} />

					<div>
						<div><strong>Example Context</strong></div>
						<pre>
							{JSON.stringify(this.props.type.test_context, null, 4)}
						</pre>
					</div>

					<Forms.FormInput label="Subject" name="subject_template" required={true} value={this.state.template.subject_template} onChange={this.onChange} />
						<Forms.TextArea label="HTML Body" name="body_html_template" value={this.state.template.body_html_template} onChange={this.onChange} rows="15" />
					<Forms.TextArea label="Text Body" name="body_text_template" value={this.state.template.body_text_template} onChange={this.onChange} rows="15" />
					<div className="text-right">
						{this.state.template.id ?
							<span><button type="submit" className="btn btn-default" onClick={this.onCancel}>Cancel</button>&nbsp;</span>
							: null}
						<button type="submit" className="btn btn-primary">{this.state.template.id?"Save":"Create"}</button>
					</div>
				</form>
			</div>
		);
	}
});
