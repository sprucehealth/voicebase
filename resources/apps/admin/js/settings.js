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
			}
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
