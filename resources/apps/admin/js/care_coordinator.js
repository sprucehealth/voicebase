/* @flow */

var Accounts = require("./accounts.js");
var AdminAPI = require("./api.js");
var Forms = require("../../libs/forms.js");
var Nav = require("../../libs/nav.js");
var Perms = require("./permissions.js");
var Routing = require("../../libs/routing.js");
var Time = require("../../libs/time.js");
var Utils = require("../../libs/utils.js");
var Modals = require("../../libs/modals.js");

module.exports = {
	Page: React.createClass({displayName: "CareCoordinatorPage",
		mixins: [Routing.RouterNavigateMixin],
		menuItems: function(): any {
			var items = [];
			if (Perms.has(Perms.CareCoordinatorView)) {
				items.push({
					id: "manage",
					url: "/carecoordinator/tags/manage",
					name: "Manage Tags"
				});
				items.push({
					id: "saved_searches",
					url: "/carecoordinator/tags/saved_searches",
					name: "Saved Searches"
				});
			}
			return [items];
		},
		pages: {
			manage: function(): any {
				return <ManageCareCoordinatorTagsPage router={this.props.router} />
			},
			saved_searches: function(): any {
				return <CareCoordinatorSavedSearchesPage router={this.props.router} />
			},
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

// BEGIN: Tags Management
var AddTagModal = React.createClass({displayName: "AddTagModal",
	getInitialState: function(): any {
		return {
			error: null,
			busy: false,
		};
	},
	onAdd: function(e): any {
		e.preventDefault()
		if (this.state.tagText == null || this.state.tagText == "") {
			this.setState({error: "tag text is required"});
			return true;
		}
		if (/\s/g.test(this.state.tagText)) {
			this.setState({error: "tags should not contain white space"});
			return true;
		}
		if (this.props.existingTags.indexOf(this.state.tagText) != -1) {
			this.setState({error: "tag " + this.state.tagText + " already exists"});
			return true; 
		}
		this.setState({busy: true, error: null});
		AdminAPI.tags(this.state.tagText, false, function(success, data, error){
			if (this.isMounted()) {
				if(success) {
					if(data.tags.length == 1) {
						// If we find the tag then we know it already exists and just isn't common
						AdminAPI.updateTag(data.tags[0].id, true, function(success, data, error) {
							if (this.isMounted()) {
								if (!success) {
									this.setState({busy: false, error: error.message});
									return;
								}
								this.setState({busy: false});
								this.props.onSuccess();
								$("#add-tag-modal").modal('hide');
							}
						}.bind(this));
					} else if (data.tags.length == 0) {
						// If we didn't find the tag then we know it's pure new and should add it as common
						AdminAPI.addTag(this.state.tagText, true, function(success, data, error) {
							if (this.isMounted()) {
								if (!success) {
									this.setState({busy: false, error: error.message});
									return;
								}
								this.setState({busy: false});
								this.props.onSuccess();
								$("#add-tag-modal").modal('hide');
							}
						}.bind(this));
					} else {
						this.setState({error: "Expected only 1 tag to be returned from match. Instead found " + JSON.stringify(data)});
					}
				} else {
						this.setState({
							busy: false,
							error: error.message,
							tags: null,
						})
				}
			}
		}.bind(this));
		return true;	
	},
	onTagTextChange: function(e): void {
		e.preventDefault();
		this.setState({
			tagText: e.target.value
		});
	},
	render: function(): any {
		return (
			<Modals.ModalForm id="add-tag-modal" title="Add Tag"
				cancelButtonTitle="Cancel" submitButtonTitle="Add"
				onSubmit={this.onAdd}>

				{this.state.error ? <Utils.Alert type="danger">{this.state.error}</Utils.Alert> : null}
				{this.state.busy ? <Utils.LoadingAnimation /> : null}

				<Forms.FormInput label="Tag Text" value={this.state.tagText} onChange={this.onTagTextChange} />
			</Modals.ModalForm>
		);
	}
});

var TagActionModal = React.createClass({displayName: "TagActionModal",
	getInitialState: function(): any {
		return {
			error: null,
			busy: false,
		};
	},
	onDelete: function(e): any {
		e.preventDefault()
		this.setState({busy: true, error: null});
		AdminAPI.updateTag(this.props.tag.id, false, function(success, data, error) {
			if (this.isMounted()) {
				if (!success) {
					this.setState({busy: false, error: error.message});
					return;
				}
				this.setState({busy: false});
				this.props.onSuccess();
				$("#tag-action-modal").modal('hide');
			}
		}.bind(this));
		return true;
	},
	render: function(): any {
		return (
			<Modals.ModalForm id="tag-action-modal" title={this.props.tag ? "Delete Tag: \"" + this.props.tag.text + "\"?": ""}
				cancelButtonTitle="Cancel" submitButtonTitle="Delete"
				onSubmit={this.onDelete}>

				{this.state.error ? <Utils.Alert type="danger">{this.state.error}</Utils.Alert> : null}
			</Modals.ModalForm>
		);
	}
});

var ManageCareCoordinatorTagsPage = React.createClass({displayName: "ManageCareCoordinatorTagsPage",
	mixins: [Routing.RouterNavigateMixin],
	getInitialState: function(): any {
		return {
			busy: true,
			error: null,
			tags: null,
		};
	},
	componentWillMount: function(): any {
		document.title = "Care Coordinator | Manage Tags";
		this.fetchTags();
	},
	fetchTags: function(): void {
		AdminAPI.tags("", true, function(success, data, error){
			if (this.isMounted()) {
				if(success) {
					this.setState({
						tags: data.tags,
						busy: false,
						error: null,
					})
				} else {
					this.setState({
						busy: false,
						error: error.message,
						tags: null,
					})
				}
			}
		}.bind(this));
	},
	setTagForAction: function(tag): void {
		this.setState({
			actingTag: tag,
		});
	},
	render: function(): any {
		return (
			<div className="container-flow" style={{marginTop: 10}}>
				{Perms.has(Perms.CareCoordinatorEdit) ? <AddTagModal onSuccess={this.fetchTags} existingTags={this.state.tags ? this.state.tags.map(function(tag){return tag.text}) : []}/> : null}
				{Perms.has(Perms.CareCoordinatorEdit) ? <TagActionModal onSuccess={this.fetchTags} tag={this.state.actingTag} /> : null}
				<h1>Available Tags</h1>
				{Perms.has(Perms.CareCoordinatorEdit) ? <div className="pull-right"><button className="btn btn-default" data-toggle="modal" data-target="#add-tag-modal">+</button></div> : null}
				{this.state.busy ? <Utils.LoadingAnimation /> : null}
				{this.state.error ? <Utils.Alert type="danger">{this.state.error}</Utils.Alert> : null}
				{(this.state.tags && Perms.has(Perms.CareCoordinatorView)) ? <TagManageList router={this.props.router} tags={this.state.tags} setTagForAction={this.setTagForAction}/> : null}
			</div>
		);
	}
});

var TagManageList = React.createClass({displayName: "TagManageList",
	mixins: [Routing.RouterNavigateMixin],
	getInitialState: function(): any {
		return {};
	},
	render: function(): any {
		return (
			<div style={{marginTop: 10}}>
				<ul className="tags">{this.props.tags.map(function(tag){
					return <TagManageListItem key={tag.id} router={this.props.router} tag={tag} setTagForAction={this.props.setTagForAction}/>
				}.bind(this))}</ul>
			</div>
		);
	}
});

var TagManageListItem = React.createClass({displayName: "TagManageListItem",
	mixins: [Routing.RouterNavigateMixin],
	getInitialState: function(): any {
		return {};
	},
	showTagActionModal: function(actionTag): void {
		this.props.setTagForAction(actionTag);
		$("#tag-action-modal").modal('show');
	},
	render: function(): any {
		return (<li><a onClick={this.showTagActionModal.bind(this, this.props.tag)} href="#">{this.props.tag.text}</a></li>);
	}
});
// END: Tags Management

// BEGIN: Saved Search Management
var AddSavedSearchModal = React.createClass({displayName: "AddSavedSearchModal",
	getInitialState: function(): any {
		return {
			error: null,
			busy: false,
		};
	},
	onAdd: function(e): any {
		e.preventDefault()
		if (!this.state.savedSearchTitleText || !this.state.savedSeearchQueryText) {
			this.setState({error: "title and query required"});
			return true;
		}
		if (this.state.savedSearchTitleText.length > 50) {
			this.setState({error: "title cannot be more than 50 characters"});
			return true; 
		}
		if (this.props.existingSavedSearchTitles.indexOf(this.state.savedSearchTitleText) != -1) {
			this.setState({error: "title " + this.state.savedSearchTitleText + " already exists"});
			return true; 
		}
		AdminAPI.addTagSearch(this.state.savedSearchTitleText, this.state.savedSeearchQueryText, function(success, data, error){
			if (this.isMounted()) {
				if(success) {
					this.setState({error: null});
					$("#add-saved-search-modal").modal('hide');
					this.props.onSuccess();
				} else {
					this.setState({
						busy: false,
						error: error.message,
					});
				}
			}
		}.bind(this));
		return true;
	},
	onTitleTextChange: function(e): void {
		e.preventDefault();
		this.setState({
			savedSearchTitleText: e.target.value
		});
	},
	onQueryTextChange: function(e): void {
		e.preventDefault();
		this.setState({
			savedSeearchQueryText: e.target.value
		});
	},
	render: function(): any {
		return (
			<Modals.ModalForm id="add-saved-search-modal" title="Add Saved Search"
				cancelButtonTitle="Cancel" submitButtonTitle="Add"
				onSubmit={this.onAdd}>

				{this.state.error ? <Utils.Alert type="danger">{this.state.error}</Utils.Alert> : null}
				{this.state.busy ? <Utils.LoadingAnimation /> : null}

				<Forms.FormInput label="Title" value={this.state.savedSearchTitleText} onChange={this.onTitleTextChange} />
				<Forms.FormInput label="Query" value={this.state.savedSeearchQueryText} onChange={this.onQueryTextChange} />
			</Modals.ModalForm>
		);
	}
});

var CareCoordinatorSavedSearchesPage = React.createClass({displayName: "CareCoordinatorSavedSearchesPage",
	mixins: [Routing.RouterNavigateMixin],
	getInitialState: function(): any {
		return {
			busy: true,
			savedSearches: null,
			error: null,
		};
	},
	componentWillMount: function(): void	{
		document.title = "Care Coordinator | Saved Searches";
		this.fetchSavedSearches();
	},
	fetchSavedSearches: function(): void {
		AdminAPI.savedTagSearches(function(success, data, error){
			if (this.isMounted()) {
				if(success) {
					this.setState({
						savedSearches: data.saved_searches,
						busy: false,
						error: null,
					});
				} else {
					this.setState({
						busy: false,
						error: error.message,
						savedSearches: null,
					});
				}
			}
		}.bind(this));
	},
	onError: function(error): void {
		this.setState({
			error: error.message,
		});
	},
	render: function(): any {
		return (
			<div className="container-flow" style={{marginTop: 10}}>
				{Perms.has(Perms.CareCoordinatorEdit) ? <AddSavedSearchModal onSuccess={this.fetchSavedSearches} existingSavedSearchTitles={this.state.savedSearches ? this.state.savedSearches.map(function(ss){return ss.title}) : []}/> : null}
				<h1>Saved Searches</h1>
				{Perms.has(Perms.CareCoordinatorEdit) ? <div className="pull-right"><button className="btn btn-default" data-toggle="modal" data-target="#add-saved-search-modal">+</button></div> : null}
				{this.state.busy ? <Utils.LoadingAnimation /> : null}
				{(this.state.savedSearches && Perms.has(Perms.CareCoordinatorView))? <SavedSearchList router={this.props.router} savedSearches={this.state.savedSearches} onDelete={this.fetchSavedSearches} onError={this.onError}/> : null}
			</div>
		);
	}
});

var SavedSearchList = React.createClass({displayName: "SavedSearchList",
	mixins: [Routing.RouterNavigateMixin],
	getInitialState: function(): any {
		return {};
	},
	render: function(): any {
		return (
			<div style={{marginTop: 10}}>
				<table className="table">
					<thead><tr><th>Title</th><th>Query</th><th>Created</th>{Perms.has(Perms.CareCoordinatorEdit) ? <th></th> : null}</tr></thead>
					<tbody>
						{this.props.savedSearches.map(function(savedSearch){return <SavedSearchListItem key={savedSearch.id} router={this.props.router} savedSearch={savedSearch} onDelete={this.props.onDelete} onError={this.props.onError} />
						}.bind(this))}
					</tbody>
				</table>
			</div>
		);
	}
});

var SavedSearchListItem = React.createClass({displayName: "SavedSearchListItem",
	mixins: [Routing.RouterNavigateMixin],
	getInitialState: function(): any {
		return {};
	},
	onDelete: function(savedSearch): void {
		this.setState({
			busy: true,
		});
		AdminAPI.deleteTagSearch(savedSearch.id, function(success, data, error){
			if (this.isMounted()) {
				this.setState({
					busy: false,
				});
				if(success) {
					this.props.onDelete();
				} else {
					this.props.onError(error);
				}
			}
		}.bind(this));
	},
	render: function(): any {
		var key = 0;
		var rowContents = [<td key={key++}>{this.props.savedSearch.title}</td>,<td key={key++}>{this.props.savedSearch.query}</td>,<td key={key++}>{(new Date(this.props.savedSearch.created_epoch * 1000)).toString()}</td>]
		if(Perms.has(Perms.CareCoordinatorEdit)){
			rowContents.push(<td key={key++}><a href="#" onClick={this.onDelete.bind(this, this.props.savedSearch)}><span className="glyphicon glyphicon-remove" /></a></td>)
		}
		if(this.state.busy) {
			rowContents.push(<td key={key++}><Utils.LoadingAnimation /></td>)
		}
		return (<tr>{rowContents}</tr>);
	}
});
// END: Saved Search Management