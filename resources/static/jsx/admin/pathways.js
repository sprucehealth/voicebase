/** @jsx React.DOM */

var AdminAPI = require("./api.js");
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
