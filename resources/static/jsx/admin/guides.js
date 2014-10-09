/** @jsx React.DOM */

var AdminAPI = require("./api.js");
var Forms = require("../forms.js");
var Modals = require("../modals.js");
var Nav = require("../nav.js");
var Routing = require("../routing.js");
var Utils = require("../utils.js");

module.exports = {
	Guides: React.createClass({displayName: "Guides",
		menuItems: [[
			{
				id: "resources",
				url: "/admin/guides/resources",
				name: "Resource Guides"
			},
			{
				id: "rx",
				url: "/admin/guides/rx",
				name: "RX Guides"
			}
		]],
		getDefaultProps: function() {
			return {
				guideID: null
			}
		},
		resources: function() {
			if (this.props.guideID != null) {
				return <ResourceGuide router={this.props.router} guideID={this.props.guideID} />;
			}
			return <ResourceGuideList router={this.props.router} />;
		},
		rx: function() {
			if (this.props.guideID != null) {
				return <RXGuide router={this.props.router} ndc={this.props.guideID} />;
			}
			return <RXGuideList router={this.props.router} />;
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

var ResourceGuide = React.createClass({displayName: "ResourceGuide",
	mixins: [Routing.RouterNavigateMixin],
	getInitialState: function() {
		return {
			guide: {},
			sections: [],
			error: ""
		};
	},
	componentWillMount: function() {
		AdminAPI.resourceGuide(this.props.guideID, function(success, data, error) {
			if (this.isMounted()) {
				if (success) {
					document.title = data.title + " | Resources | Guides | Spruce Admin";
					data.layout_json = JSON.stringify(data.layout, null, 4);
					this.setState({guide: data});
				} else {
					// TODO
					alert("Failed to get resource guide: " + error.message);
				}
			}
		}.bind(this));
		AdminAPI.resourceGuidesList(false, true, function(success, data, error) {
			if (this.isMounted()) {
				if (success) {
					this.setState({sections: data.sections});
				} else {
					// TODO
					alert("Failed to get sections: " + error.message);
				}
			}
		}.bind(this));
	},
	onChange: function(e) {
		var guide = this.state.guide;
		var val = e.target.value;
		// Make sure to maintain types
		if (typeof guide[e.target.name] == "number") {
			val = Number(val);
		}
		guide[e.target.name] = val;
		this.setState({error: "", guide: guide});
		return false;
	},
	onSubmit: function() {
		try {
			var guide = this.state.guide;
			var js = JSON.parse(guide.layout_json);
			guide.layout = js;
			this.setState({error: "", guide: guide});
		} catch (err) {
			this.setState({error: "Invalid layout: " + err.message});
			return false;
		};

		AdminAPI.updateResourceGuide(this.props.guideID, this.state.guide, function(success, data, error) {
			if (this.isMounted()) {
				if (!success) {
					// TODO
					alert("Failed to save resource guide: " + error.message);
				}
			}
		}.bind(this));
		return false;
	},
	render: function() {
		var sectionOptions = this.state.sections.map(function(s) {
			return {value: s.id, name: s.title};
		})
		return (
			<div className="resource-guide-edit">
				<h2><img src={this.state.guide.photo_url} width="32" height="32" /> {this.state.guide.title}</h2>

				<form role="form" onSubmit={this.onSubmit} method="PUT">
					<div className="row">
						<div className="col-md-2">
							<Forms.FormSelect name="section_id" label="Section" value={this.state.guide.section_id} opts={sectionOptions} onChange={this.onChange} />
						</div>
						<div className="col-md-2">
							<Forms.FormInput name="ordinal" type="number" required label="Ordinal" value={this.state.guide.ordinal} onChange={this.onChange} />
						</div>
						<div className="col-md-8">
							<Forms.FormInput name="photo_url" type="url" required label="Photo URL" value={this.state.guide.photo_url} onChange={this.onChange} />
						</div>
					</div>
					<div>
						<Forms.FormInput name="title" type="text" required label="Title" value={this.state.guide.title} onChange={this.onChange} />
					</div>
					<div>
						<Forms.TextArea name="layout_json" required label="Layout" value={this.state.guide.layout_json} rows="30" onChange={this.onChange} />
					</div>
					<div className="text-right">
						{this.state.error ? <Utils.Alert type="danger">{this.state.error}</Utils.Alert> : null}
						<button type="submit" className="btn btn-primary">Save</button>
					</div>
				</form>
			</div>
		);
	}
});

var ResourceGuideList = React.createClass({displayName: "ResourceGuideList",
	mixins: [Routing.RouterNavigateMixin],
	getInitialState: function() {
		return {sections: []};
	},
	componentWillMount: function() {
		document.title = "Resources | Guides | Spruce Admin";
		this.updateList();
	},
	updateList: function() {
		AdminAPI.resourceGuidesList(false, false, function(success, data, error) {
			if (this.isMounted()) {
				if (success) {
					var sections = data.sections;
					for(var i = 0; i < sections.length; i++) {
						var s = sections[i];
						s.guides = data.guides[s.id];
					}
					this.setState({sections: sections});
				} else {
					// TODO
					alert("Failed to get resource guides: " + error.message);
				}
			}
		}.bind(this));
	},
	onImport: function(e) {
		e.preventDefault();
		var formData = new FormData(e.target);
		AdminAPI.resourceGuidesImport(formData, function(success, data, error) {
			if (!success) {
				// TODO
				alert("Failed to import resource guides: " + error.message);
				return;
			}
			this.updateList();
		}.bind(this));
		return false;
	},
	onExport: function(e) {
		e.preventDefault();
		AdminAPI.resourceGuidesExport(function(success, data, error) {
			if (this.isMounted()) {
				if (success) {
					var pom = document.createElement('a');
					pom.setAttribute('href', 'data:application/octet-binary;charset=utf-8,' + encodeURIComponent(data));
					pom.setAttribute('download', "resource_guides.json");
					pom.click();
				} else {
					// TODO
					alert("Failed to get resource guides: " + error.message);
				}
			}
		}.bind(this));
		return false;
	},
	render: function() {
		var t = this;
		var createSection = function(section) {
			return (
				<div key={section.id}>
					<Section router={this.props.router} section={section} />
				</div>
			);
		}.bind(this);
		return (
			<div>
				<Modals.ModalForm id="import-resource-guides-modal" title="Import Resource Guides" cancelButtonTitle="Cancel" submitButtonTitle="Import" onSubmit={this.onImport}>
					<input required type="file" name="json" />
				</Modals.ModalForm>
				<div className="pull-right">
					<button className="btn btn-default" data-toggle="modal" data-target="#import-resource-guides-modal">Import</button>
					&nbsp;
					<button className="btn btn-default" onClick={this.onExport}>Export</button>
				</div>
				<div>{this.state.sections.map(createSection)}</div>
			</div>
		);
	}
});

var Section = React.createClass({displayName: "Section",
	mixins: [Routing.RouterNavigateMixin],
	getInitialState: function() {
		return {editing: false};
	},
	render: function() {
		var createGuideItem = function(guide) {
			guide.key = guide.id;
			guide.router = this.props.router;
			return GuideListItem(guide);
		}.bind(this);
		var title;
		if (this.state.editing) {
			title = <input type="text" className="form-control section-name" value={this.props.section.title} />;
		} else {
			title = <h4>{this.props.section.title}</h4>;
		}
		return (
			<div className="section">
				{title}
				{this.props.section.guides.map(createGuideItem)}
			</div>
		);
	}
});

var GuideListItem = React.createClass({displayName: "GuideListItem",
	mixins: [Routing.RouterNavigateMixin],
	render: function() {
		return (
			<div key={this.props.id} className="item">
				<img src={this.props.photo_url} width="32" height="32" />
				&nbsp;<a href={"resources/"+this.props.id} onClick={this.onNavigate}>{this.props.title}</a>
			</div>
		);
	}
});

var RXGuide = React.createClass({displayName: "RXGuide",
	mixins: [Routing.RouterNavigateMixin],
	getInitialState: function() {
		return {"guide": []}
	},
	componentWillMount: function() {
		AdminAPI.rxGuide(this.props.ndc, true, function(success, data, error) {
			if (this.isMounted()) {
				if (success) {
					document.title = this.props.ndc + " | RX | Guides | Spruce Admin";
					this.setState({guide: data.guide, html: data.html});
				} else {
					// TODO
					alert("Failed to get rx guide: " + error.message);
				}
			}
		}.bind(this));
	},
	render: function() {
		return (
			<div className="rxguide">
				<div dangerouslySetInnerHTML={{__html: this.state.html}}></div>
			</div>
		);
	}
});

var RXGuideList = React.createClass({displayName: "RXGuideList",
	mixins: [Routing.RouterNavigateMixin],
	getInitialState: function() {
		return {"guides": []}
	},
	componentWillMount: function() {
		document.title = "RX | Guides | Spruce Admin";
		this.updateList();
	},
	updateList: function() {
		AdminAPI.rxGuidesList(function(success, data, error) {
			if (this.isMounted()) {
				if (success) {
					this.setState({guides: data});
				} else {
					// TODO
					alert("Failed to get rx guides: " + error.message);
				}
			}
		}.bind(this));
	},
	onImport: function(e) {
		e.preventDefault();
		var formData = new FormData(e.target);
		AdminAPI.rxGuidesImport(formData, function(success, data, error) {
			if (!success) {
				// TODO
				alert("Failed to import rx guides: " + error.message);
				return;
			}
			this.updateList();
		}.bind(this));
		return false;
	},
	render: function() {
		return (
			<div className="rx-guide-list">
				<Modals.ModalForm id="import-rx-guides-modal" title="Import RX Guides" cancelButtonTitle="Cancel" submitButtonTitle="Import" onSubmit={this.onImport}>
					<input required type="file" name="csv" />
				</Modals.ModalForm>
				<div className="pull-right">
					<button className="btn btn-default" data-toggle="modal" data-target="#import-rx-guides-modal">Import</button>
				</div>

				<h2>RX Guides</h2>
				{this.state.guides.map(function(guide) {
					return <div key={guide.NDC} className="rx-guide">
						<a href={"/admin/guides/rx/" + guide.NDC} onClick={this.onNavigate}>{guide.Name + " (" + guide.NDC + ")"}</a>
					</div>
				}.bind(this))}
			</div>
		);
	}
});
