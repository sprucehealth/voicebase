/* @flow */

import * as AdminAPI from "./api"
import * as Forms from "../../libs/forms"
import * as Nav from "../../libs/nav"
import * as Perms from "./permissions"
import * as React from "react"
import * as Routing from "../../libs/routing"

module.exports = {
	AccountList: React.createClass({displayName: "AccountList",
		mixins: [Routing.RouterNavigateMixin],
		getInitialState: function(): any {
			return {
				query: "",
				results: null
			};
		},
		componentWillMount: function() {
			document.title = "Accounts | Spruce Admin";
		},
		search: function(q: string): void {
			if (q == "") {
				this.setState({results: null})
			} else {
				AdminAPI.searchAdmins(q, (success, res, error) => {
					if (this.isMounted()) {
						if (!success) {
							// TODO
							alert(error.message);
							return;
						}
						this.setState({results: res.accounts || []});
					}
				});
			}
		},
		onSearchSubmit: function(e: Event): void {
			e.preventDefault();
			this.search(this.state.query);
		},
		onQueryChange: function(e: any): void {
			this.setState({query: e.target.value});
		},
		render: function(): any {
			return (
				<div className="container accounts-search">
					<div className="row">
						<div className="col-md-3">&nbsp;</div>
						<div className="col-md-6">
							<h2>Search admin accounts</h2>
							<form onSubmit={this.onSearchSubmit}>
								<div className="form-group">
									<input required autofocus type="email" className="form-control" name="q" value={this.state.query} onChange={this.onQueryChange} />
								</div>
								<button type="submit" className="btn btn-primary btn-lg center-block">Search</button>
							</form>
						</div>
						<div className="col-md-3">&nbsp;</div>
					</div>

					{this.state.results ? <AccountSearchResults
						router = {this.props.router}
						results = {this.state.results} />
					: null}
				</div>
			);
		}
	}),

	Account: React.createClass({displayName: "Account",
		menuItems: [[
			{
				id: "permissions",
				url: "permissions",
				name: "Permissions"
			}
		]],
		pages: {
			permissions: function(): any {
				return <AccountPermissionsPage router={this.props.router} account={this.state.account} />;
			},
		},
		getInitialState: function(): any {
			return {
				account: null
			};
		},
		componentWillMount: function() {
			AdminAPI.adminAccount(this.props.accountID, (success, data, error) => {
				if (this.isMounted()) {
					if (!success) {
						// TODO
						alert("Failed to fetch account: " + error.message);
						return;
					}
					this.setState({account: data.account});
				}
			});
		},
		render: function(): any {
			if (this.state.account == null) {
				// TODO
				return <div>LOADING</div>;
			}
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

var AccountSearchResults = React.createClass({displayName: "AccountSearchResults",
	mixins: [Routing.RouterNavigateMixin],
	render: function() {
		if (this.props.results.length == 0) {
			return (<div className="no-results">No Results</div>);
		}

		var results = this.props.results.map(res => {
			return (
				<div className="row" key={res.id}>
					<div className="col-md-3">&nbsp;</div>
					<div className="col-md-6">
						<AccountSearchResult result={res} router={this.props.router} />
					</div>
					<div className="col-md-3">&nbsp;</div>
				</div>
			);
		})

		return (
			<div className="search-results">{results}</div>
		);
	}
});

var AccountSearchResult = React.createClass({displayName: "AccountSearchResult",
	mixins: [Routing.RouterNavigateMixin],
	render: function() {
		return (
			<a href={"/accounts/"+this.props.result.id+"/permissions"} onClick={this.onNavigate}>{this.props.result.email}</a>
		);
	}
});

var AccountPermissionsPage = React.createClass({displayName: "AccountPermissionsPage",
	mixins: [Routing.RouterNavigateMixin],
	getInitialState: function() {
		return {
			groups: [],
			permissions: []
		};
	},
	componentWillMount: function() {
		this.loadGroups();
		this.loadPermissions();
	},
	loadGroups: function() {
		AdminAPI.adminGroups(this.props.account.id, (success, data, error) => {
			if (this.isMounted()) {
				if (!success) {
					// TODO
					alert("Failed to fetch account groups: " + error.message);
					return;
				}
				this.setState({groups: data.groups.sort((a, b) => { return a.name > b.name; })});
			}
		});
	},
	loadPermissions: function() {
		AdminAPI.adminPermissions(this.props.account.id, (success, data, error) => {
			if (this.isMounted()) {
				if (!success) {
					// TODO
					alert("Failed to fetch account permissions: " + error.message);
					return;
				}
				this.setState({permissions: data.permissions.sort((a, b) => { return a > b })});
			}
		});
	},
	updateGroups: function(updates) {
		AdminAPI.updateAdminGroups(this.props.account.id, updates, (success, data, error) => {
			if (this.isMounted()) {
				if (!success) {
					// TODO
					alert("Failed to update permissions: " + error.message);
					return;
				}
				this.loadGroups();
				this.loadPermissions();
			}
		});
	},
	onRemoveGroup: function(group) {
		var updates = {};
		updates[group] = false;
		this.updateGroups(updates);
	},
	onAddGroup: function(group) {
		var updates = {};
		updates[group] = true;
		this.updateGroups(updates);
	},
	render: function() {
		return (
			<div>
				<h2>{this.props.account.email}</h2>
				<h3>Groups</h3>
				<AccountGroups allowEdit={Perms.has(Perms.AdminAccountsEdit)} onAdd={this.onAddGroup} onRemove={this.onRemoveGroup} groups={this.state.groups} />
				<h3>Permissions</h3>
				<AccountPermissions permissions={this.state.permissions} />
			</div>
		);
	}
});

var AccountGroups = React.createClass({displayName: "AccountGroups",
	getDefaultProps: function() {
		return {
			groups: [],
			allowEdit: false
		};
	},
	getInitialState: function() {
		return {
			adding: false,
			addingValue: null,
			availableGroups: []
		};
	},
	componentWillMount: function() {
		AdminAPI.availableGroups(true, (success, data, error) => {
			if (this.isMounted()) {
				if (!success) {
					// TODO
					alert("Failed to fetch available groups: " + error.message);
					return;
				}
				var groupOptions = data.groups.map(g => { return {value: g.id, name: g.name} });
				this.setState({
					availableGroups: data.groups,
					groupOptions: groupOptions,
					addingValue: groupOptions[0].value
				});
			}
		});
	},
	onAdd: function(e) {
		e.preventDefault();
		this.setState({adding: true});
	},
	onChange: function(e) {
		this.setState({addingValue: e.target.value});
	},
	onCancel: function(e) {
		e.preventDefault();
		this.setState({adding: false});
	},
	onSubmit: function(e) {
		e.preventDefault();
		this.props.onAdd(this.state.addingValue);
		this.setState({adding: false, addingValue: this.state.groupOptions[0].value});
	},
	onRemove: function(group) {
		this.props.onRemove(group);
		return false;
	},
	render: function() {
		return (
			<div className="groups">
				{this.props.groups.map((group) => {
					return (
						<div key={group.id}>
							{this.props.allowEdit ? <a href="#" onClick={this.onRemove.bind(this, group.id)}><span className="glyphicon glyphicon-remove" /></a> : null} {group.name}
						</div>
					);
				})}
				{this.state.adding ?
					<div>
						<form onSubmit={this.onSubmit}>
							<Forms.FormSelect onChange={this.onChange} value={this.addingValue} opts={this.state.groupOptions} />
							<button onClick={this.onCancel} className="btn btn-default">Cancel</button>
							&nbsp;<button type="submit" className="btn btn-default">Save</button>
						</form>
					</div> : null}
				{this.props.allowEdit && !this.state.adding ?
					<div><a href="#" onClick={this.onAdd}><span className="glyphicon glyphicon-plus" /></a></div> : null}
			</div>
		);
	}
});

var AccountPermissions = React.createClass({displayName: "AccountPermissions",
	getDefaultProps: function() {
		return {
			permissions: []
		};
	},
	render: function() {
		return (
			<div className="permissions">
				{this.props.permissions.map(perm => {
					return <div key={perm}>{perm}</div>;
				})}
			</div>
		);
	}
});
