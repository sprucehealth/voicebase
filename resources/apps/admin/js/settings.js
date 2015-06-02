/* @flow */

var Accounts = require("./accounts.js");
var AdminAPI = require("./api.js");
var Forms = require("../../libs/forms.js");
var Nav = require("../../libs/nav.js");
var Perms = require("./permissions.js");
var Routing = require("../../libs/routing.js");
var Time = require("../../libs/time.js");
var Utils = require("../../libs/utils.js");

module.exports = {
	Page: React.createClass({displayName: "SettingsPage",
		mixins: [Routing.RouterNavigateMixin],
		pages: {
			accounts: function(): any {
				return <Accounts.AccountList router={this.props.router} />
			},
			cfg: function(): any {
				return <Cfg router={this.props.router} />
			},
			schedmsg: function(): any {
				return <ScheduledMessageTemplates router={this.props.router} />
			},
			email: function(): any {
				return <Email router={this.props.router} />
			},
		},
		menuItems: function(): any {
			var items = [];
			if (Perms.has(Perms.AdminAccountsView)) {
				items.push({
					id: "accounts",
					url: "accounts",
					name: "Accounts"
				})
			}
			if (Perms.has(Perms.CfgView)) {
				items.push({
					id: "cfg",
					url: "cfg",
					name: "REST API Config"
				});
			}
			if (Perms.has(Perms.AppMessageTemplatesView)) {
				items.push({
					id: "schedmsg",
					url: "schedmsg",
					name: "Scheduled Message Templates"
				});
			}
			if (Perms.has(Perms.EmailEdit)) {
				items.push({
					id: "email",
					url: "email",
					name: "Email Testing"
				});
			}
			return [items];
		},
		componentWillMount: function() {
			if (!this.props.page) {
				this.props.router.navigate("/settings/accounts", {replace: true});
			}
		},
		render: function(): any {
			if (!this.props.page) {
				return null;
			}
			return (
				<div>
					<Nav.LeftNav router={this.props.router} items={this.menuItems()} currentPage={this.props.page}>
						{this.pages[this.props.page].bind(this)()}
					</Nav.LeftNav>
				</div>
			);
		}
	})
};

var Cfg = React.createClass({displayName: "Cfg",
	mixins: [Routing.RouterNavigateMixin],
	getInitialState: function() {
		return {
			error: null,
			busy: false,
			snapshot: {},
			modified: false,
			updates: {},
			updateErrors: {},
			defs: {}
		};
	},
	componentWillMount: function() {
		document.title = "REST API Cfg | Spruce Admin";
		this.update();
	},
	update: function() {
		this.setState({busy: true, error: null});
		AdminAPI.cfg(function(success, data, error) {
			if (this.isMounted()) {
				if (!success) {
					this.setState({
						error: error.message,
						busy: false,
					});
					return;
				}
				this.setState({
					snapshot: data.snapshot,
					defs: data.defs,
					busy: false,
				});
			}
		}.bind(this));
	},
	handleChange: function(name, value) {
		this.state.updates[name] = value;
		if (this.state.defs[name].type == "duration") {
			var d = Time.parseDuration(value);
			if (d.err) {
				this.state.updateErrors[name] = d.err;
			} else if (this.state.updateErrors[name]) {
				delete this.state.updateErrors[name];
			}
		}
		this.setState({modified: true, updates: this.state.updates, updateErrors: this.state.updateErrors});
	},
	handleCancel: function() {
		this.setState({modified: false, updates: {}, updateErrors: {}});
	},
	handleSave: function() {
		if (this.state.modified) {
			this.setState({busy: true, error: null});
			var updates = {};
			for(var name in this.state.updates) {
				var v = this.state.updates[name];
				var d = this.state.defs[name];
				if (d.type == "duration") {
					var d = Time.parseDuration(v);
					if (d.err) {
						// Shouldn't happen during save but handle it just in case
						this.setState({busy: false, error: "Invalid value for " + name + ": " + d.err});
						return
					}
					v = d.d;
				}
				updates[name] = v;
			}
			AdminAPI.updateCfg(updates, function(success, data, error) {
				if (this.isMounted()) {
					if (!success) {
						this.setState({
							error: error.message,
							busy: false,
						});
						return;
					}
					this.handleCancel();
					this.setState({
						snapshot: data.snapshot,
						defs: data.defs,
						busy: false,
					});
				}
			}.bind(this));
		}
	},
	render: function(): any {
		var rows = [];
		var hasError = false;
		for(var name in this.state.defs) {
			var v = null;
			if (typeof this.state.snapshot[name] != "undefined") {
				v = this.state.snapshot[name];
			}
			var d = this.state.defs[name];
			if (v == null) {
				v = d.default;
			}
			var e = this.state.updateErrors[name] || null;
			if (this.state.modified && typeof this.state.updates[name] != "undefined") {
				v = this.state.updates[name];
			} else if (d.type == "duration") {
				v = Time.formatDuration(v);
			}
			if (e) {
				hasError = true;
			}
			rows.push(
				<CfgRow
					key = {"cfg-"+name}
					def = {d}
					val = {v}
					err = {e}
					onChange = {this.handleChange} />);
		}
		return (
			<div>
				<table className="table">
					<thead>
						<tr>
							<th>Name</th>
							<th>Value</th>
							<th>Type</th>
							<th>Description</th>
						</tr>
					</thead>
					<tbody>
						{rows}
					</tbody>
				</table>
				{this.state.busy ? <Utils.LoadingAnimation /> : null}
				{this.state.error ? <Utils.Alert type="danger">{this.state.error}</Utils.Alert> : null}
				{this.state.modified ?
					<div className="text-right">
						<button
							className = "btn btn-default"
							onClick = {this.handleCancel}>Cancel</button>
						{" "}<button
							className = "btn btn-primary"
							disabled = {hasError}
							onClick = {this.handleSave}>Save</button>
					</div>
					: null}
			</div>
		);
	}
});

var cfgInputTypes = {
	"string": "text",
	"bool": "text",
	"int": "number",
	"float": "number",
	"duration": "text",
};

var CfgRow = React.createClass({displayName: "CfgRow",
	handleChange: function(e: any) {
		e.preventDefault();
		this.props.onChange(this.props.def.name, e.target.value);
	},
	handleChangeBool: function(e: any, checked: bool) {
		this.props.onChange(this.props.def.name, checked);
	},
	render: function(): any {
		var input: any;
		if (this.props.def.type == "bool") {
			input = <Forms.Checkbox
				checked = {this.props.val}
				onChange = {this.handleChangeBool} />
		} else if (this.props.def.multi) {
			input = <Forms.TextArea
				value = {this.props.val}
				onChange = {this.handleChange} />
		} else {
			input = <Forms.FormInput
				type = {cfgInputTypes[this.props.def.type] || "text"}
				value = {this.props.val}
				onChange = {this.handleChange} />
		}
		return (
			<tr>
				<td>{this.props.def.name}</td>
				<td>
					{input}
					{this.props.err ? <Utils.Alert type="danger">{this.props.err}</Utils.Alert> : null}
				</td>
				<td>{this.props.def.type}</td>
				<td>{this.props.def.description}</td>
			</tr>
		);
	}
});

var ScheduledMessageTemplates = React.createClass({displayName: "ScheduledMessageTemplates",
	getInitialState: function() {
		return {
			error: null,
			busy: false,
			templates: [],
			selectedEvent: null,
			edited: false,
		}
	},
	componentWillMount: function() {
		AdminAPI.listScheduledMessageTemplates(function(success, data, error) {
			if (this.isMounted()) {
				if (!success) {
					this.setState({
						error: error.message,
						busy: false,
					});
					return;
				}
				var ev = this.state.selectedEvent;
				if (ev == null) {
					if (data.length != 0) {
						ev = data[0].event;
					} else {
						ev = null;
					}
				}
				this.setState({
					templates: data,
					defs: data.defs,
					busy: false,
					selectedEvent: ev,
					error: null,
				});
			}
		}.bind(this));
	},
	handleSelectEvent: function(e) {
		e.preventDefault();
		this.setState({selectedEvent: e.target.value});
	},
	handleChange: function(e) {
		e.preventDefault();
		var t = this.selectedTemplate();
		if (t == null) {
			console.error("Edit while no selected template.");
			return;
		}
		t[e.target.name] = e.target.value;
		this.setState({templates: this.state.templates, edited: true});
	},
	handleSubmit: function(e) {
		e.preventDefault();
		var t = this.selectedTemplate();
		if (t == null) {
			console.error("Save while no selected template.");
			return;
		}
		this.setState({busy: true});
		AdminAPI.updateScheduledMessageTemplate(t.id, t, function(success, data, error) {
			if (this.isMounted()) {
				if (!success) {
					this.setState({
						error: error.message,
						busy: false,
					});
					return;
				}
				this.setState({
					busy: false,
					error: null,
					edited: false,
				});
			}
		}.bind(this));
	},
	selectedTemplate: function() {
		for(var i = 0; i < this.state.templates.length; i++) {
			var t = this.state.templates[i];
			if (t.event == this.state.selectedEvent) {
				return t;
			}
		}
		return null;
	},
	render: function(): any {
		var events = this.state.templates.map(function(t) {
			return {name: t.event, value: t.event};
		});
		var tmpl = this.selectedTemplate();
		return (
			<div>
				{this.state.busy ? <Utils.LoadingAnimation /> : null}
				{this.state.error ? <Utils.Alert type="danger">{this.state.error}</Utils.Alert> : null}
				<div>
					<Forms.FormSelect label="Event" value={this.state.selectedEvent} opts={events} onChange={this.handleSelectEvent} />
					{tmpl ?
						<form onSubmit={this.handleSubmit}>
							<Forms.FormInput label="Name" name="name" value={tmpl.name} onChange={this.handleChange} />
							<Forms.FormInput label="Period" name="scheduled_period" type="number" value={tmpl.scheduled_period} onChange={this.handleChange} />
							<Forms.TextArea label="Message" name="message" value={tmpl.message} rows={20} onChange={this.handleChange} />
							{this.state.edited && Perms.has(Perms.AppMessageTemplatesEdit) ?
								<div className="text-right">
									<button className="btn btn-primary" type="submit">Save</button>
								</div>
							: null}
						</form>
					: null}
				</div>
			</div>
		);
	}
});

// TODO(samuelks): move this to the marketing section once that exists
var Email = React.createClass({displayName: "Email",
	mixins: [Routing.RouterNavigateMixin],
	getInitialState: function() {
		return {
			error: null,
			busy: false,
			sent: false,
			emailType: ""
		};
	},
	componentWillMount: function() {
		document.title = "Email | Spruce Admin";
	},
	handleChangeEmailType: function(e: any) {
		e.preventDefault();
		this.setState({emailType: e.target.value});
	},
	handleSubmit: function(e: Event) {
		e.preventDefault();
		this.setState({busy: true, sent: false});
		AdminAPI.sendTestEmail(this.state.emailType, function(success, data, error) {
			if (this.isMounted()) {
				if (!success) {
					this.setState({busy: false, error: error.message});
					return;
				}
				if (data.success) {
					this.setState({busy: false, sent: true, error: null});
				} else {
					this.setState({busy: false, error: data.error});
				}
			}
		}.bind(this));
	},
	render: function(): any {
		return (
			<div>
				{this.state.busy ? <Utils.LoadingAnimation /> : null}
				{this.state.error ? <Utils.Alert type="danger">{this.state.error}</Utils.Alert> : null}
				{this.state.sent ? <Utils.Alert type="success">Sent Successfully</Utils.Alert> : null}
				<form onSubmit={this.handleSubmit}>
					<Forms.FormInput required={true} label="Type" name="type" value={this.state.emailType} onChange={this.handleChangeEmailType} />
					<div className="text-right">
						<button className="btn btn-primary" type="submit">Send</button>
					</div>
				</form>
			</div>
		);
	}
});
