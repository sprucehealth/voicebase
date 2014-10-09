/** @jsx React.DOM */

var AdminAPI = require("./api.js");
var Forms = require("../forms.js");
var Nav = require("../nav.js");
var Perms = require("./permissions.js");
var Routing = require("../routing.js");
var Utils = require("../utils.js");

module.exports = {
	Analytics: React.createClass({displayName: "Analytics",
		mixins: [Routing.RouterNavigateMixin],
		getInitialState: function() {
			return {
				reports: [],
				menuItems: this.defaultMenuItems(),
			};
		},
		defaultMenuItems: function() {
			var menuItems = [];
			if (Perms.has(Perms.AnalyticsReportsEdit)) {
				menuItems.push([
					{
						id: "query",
						url: "/admin/analytics/query",
						name: "Query"
					}
				]);
			}
			return menuItems;
		},
		componentWillMount: function() {
			// TODO: use ace editor for syntax highlighting
			// var script = document.createElement("script");
			// script.setAttribute("src", "https://cdnjs.cloudflare.com/ajax/libs/ace/1.1.3/ace.js")
			// document.head.appendChild(script);

			document.title = "Analytics | Spruce Admin";

			this.loadReports();
		},
		loadReports: function() {
			AdminAPI.listAnalyticsReports(function(success, data, error) {
				if (this.isMounted()) {
					if (!success) {
						// TODO
						alert("Failed to get reports list: " + error.message);
						return;
					}
					data = data || [];
					var repMenu = [];
					for(var i = 0; i < data.length; i++) {
						var rep = data[i];
						repMenu.push({
							id: "report-" + rep.id,
							url: "/admin/analytics/reports/" + rep.id,
							name: rep.name
						});
					}
					var menuItems = this.defaultMenuItems();
					menuItems.push(repMenu);
					this.setState({
						reports: data,
						menuItems: menuItems
					});

					if (this.props.page == "query" && !Perms.has(Perms.AnalyticsReportsEdit)) {
						this.navigate("/analytics/reports/" + data[0].id);
					}
				}
			}.bind(this));
		},
		onSaveReport: function(report) {
			this.loadReports();
		},
		query: function() {
			if (!Perms.has(Perms.AnalyticsReportsEdit)) {
				return <div></div>;
			}
			return <AnalyticsQuery router={this.props.router} />;
		},
		reports: function() {
			return <AnalyticsReport router={this.props.router} reportID={this.props.reportID} onSave={this.onSaveReport} />;
		},
		render: function() {
			// TODO: this is janky
			var currentPage = this.props.page;
			if (currentPage == "reports") {
				currentPage = "report-" + this.props.reportID;
			}
			return (
				<div>
					<Nav.LeftNav router={this.props.router} items={this.state.menuItems} currentPage={currentPage}>
						{this[this.props.page]()}
					</Nav.LeftNav>
				</div>
			);
		}
	})
};

function DownloadAnalyticsCSV(results, name) {
	// Generate CSV of the results
	var csv = results.cols.join(",");
	for(var i = 0; i < results.rows.length; i++) {
		var row = results.rows[i];
		csv += "\n" + row.map(function(v) {
			if (typeof v == "number") {
				return v;
			} else if (typeof v == "string") {
				return '"' + v.replace(/"/g, '""') + '"';
			} else {
				return '"' + v.toString().replace(/"/g, '""') + '"';
			}
		}).join(",");
	}

	name = name || "analytics";

	var pom = document.createElement('a');
	pom.setAttribute('href', 'data:application/octet-binary;charset=utf-8,' + encodeURIComponent(csv));
	pom.setAttribute('download', name + ".csv");
	pom.click();
}

var AnalyticsQuery = React.createClass({displayName: "AnalyticsQuery",
	mixins: [Routing.RouterNavigateMixin],
	getInitialState: function() {
		return {
			query: "",
			presentation: "",
			error: "",
			running: false,
			results: null
		};
	},
	query: function(q) {
		if (q == "") {
			this.setState({error: "", results: null})
		} else {
			this.setState({running: true, error: ""});
			AdminAPI.analyticsQuery(q, function(success, res, error) {
				if (this.isMounted()) {
					this.setState({running: false});
					if (!success) {
						// TODO
						alert(error.message);
						return;
					}
					if (res.error) {
						this.setState({error: res.error, results: null})
					} else {
						this.setState({
							error: "",
							results: {
								cols: res.cols,
								rows: res.rows
							}
						});
					}
				}
			}.bind(this));
		}
	},
	onSubmit: function(e) {
		e.preventDefault();
		this.query(this.state.query);
		return false;
	},
	onQueryChange: function(e) {
		this.setState({query: e.target.value});
	},
	onDownload: function(e) {
		e.preventDefault();
		DownloadAnalyticsCSV(this.state.results);
		return false;
	},
	onSave: function(e) {
		e.preventDefault();
		AdminAPI.createAnalyticsReport("New Report", this.state.query, "", function(success, reportID, error) {
			if (this.isMounted()) {
				if (!success) {
					this.setState({error: "Failed to save report: " + error.message});
					return;
				}
				this.navigate("/analytics/reports/" + reportID);
			}
		}.bind(this));
		return false;
	},
	render: function() {
		return (
			<div className="analytics">
				<div className="form">
					<div className="text-center">
						<h2>Analytics</h2>
					</div>
					<form onSubmit={this.onSubmit}>
						<Forms.TextArea tabs="true" label="Query" name="q" value={this.state.query} onChange={this.onQueryChange} rows="10" />
						<div className="text-center">
							<button className="btn btn-default" onClick={this.onSave}>Save</button>
							&nbsp;<button disabled={this.state.results ? "" : "disabled"} className="btn btn-default" onClick={this.onDownload}>Download</button>
							&nbsp;<button type="submit" className="btn btn-primary">Query</button>
						</div>
					</form>
				</div>

				{this.state.error ? <Utils.Alert type="danger">{this.state.error}</Utils.Alert> : null}

				{this.state.running ? <Utils.Alert type="info">Querying... please wait</Utils.Alert> : null}

				{this.state.results ? AnalyticsTable({
					router: this.props.router,
					data: this.state.results
				}) : null}
			</div>
		);
	}
});

var AnalyticsQueryCache = {};

var AnalyticsReport = React.createClass({displayName: "AnalyticsReport",
	mixins: [Routing.RouterNavigateMixin],
	getInitialState: function() {
		return {
			id: null,
			name: "",
			query: "",
			presentation: "",
			error: "",
			running: false,
			results: null,
			version: 1,
			editing: false
		};
	},
	componentWillMount: function() {
		this.loadReport(this.props.reportID);
	},
	componentWillReceiveProps: function(nextProps) {
		if (this.props.reportID != nextProps.reportID) {
			this.loadReport(nextProps.reportID);
		}
	},
	componentWillUpdate: function(nextProps, nextState) {
		document.analyticsData = nextState.results;
	},
	loadReport: function(id) {
		AdminAPI.analyticsReport(id, function(success, report, error) {
			if (this.isMounted()) {
				if (!success) {
					this.setState({error: "Failed to load report: " + error.message})
					return
				}
				document.title = report.name + " | Analytics | Spruce Admin";
				this.setState({
					id: report.id,
					name: report.name,
					query: report.query,
					presentation: report.presentation,
					error: "",
					results: AnalyticsQueryCache[report.id],
					editing: report.name == "New Report",
					version: this.state.version+1
				});
			}
		}.bind(this));
	},
	query: function(q) {
		if (q == "") {
			this.setState({error: "", results: null})
		} else {
			this.setState({running: true, error: ""});
			AdminAPI.analyticsQuery(q, function(success, res, error) {
				if (this.isMounted()) {
					this.updateResults(success, res, error)
				}
			}.bind(this));
		}
	},
	updateResults: function(success, res, error) {
		this.setState({running: false});
		if (!success) {
			// TODO
			alert(error.message);
			return;
		}
		if (res.error) {
			this.setState({error: res.error, results: null})
		} else {
			results = {
				cols: res.cols,
				rows: res.rows
			}
			this.setState({
				error: "",
				results: results
			});
			AnalyticsQueryCache[this.state.id] = results;
			// TODO: push changes to presentation
			// var pres = this.refs.presentation;
			// if (pres != null) {
			// 	var onUpdate = pres.getDOMNode().onUpdate;
			// }
			// TODO: for now just force the iframe to reload
			this.setState({version: this.state.version+1});
		}
	},
	onSubmit: function(e) {
		e.preventDefault();
		this.query(this.state.query);
		return false;
	},
	onNameChange: function(e) {
		this.setState({name: e.target.value});
	},
	onQueryChange: function(e) {
		this.setState({query: e.target.value});
	},
	onPresentationChange: function(e) {
		this.setState({presentation: e.target.value});
	},
	onDownload: function(e) {
		e.preventDefault();
		DownloadAnalyticsCSV(this.state.results, this.state.name);
		return false;

	},
	onSave: function(e) {
		e.preventDefault();
		AdminAPI.updateAnalyticsReport(this.props.reportID, this.state.name, this.state.query, this.state.presentation, function(success, data, error) {
			if (this.isMounted()) {
				if (!success) {
					this.setState({error: "Failed to save report: " + error.message});
					return;
				}
				if (this.props.onSave) {
					this.props.onSave({
						id: this.props.reportID,
						error: "",
						name: this.state.name,
						query: this.state.query,
						presentation: this.state.presentation,
					});
				}
				this.setState({version: this.state.version+1});
			}
		}.bind(this));
		return false;
	},
	onEdit: function(e) {
		e.preventDefault();
		this.setState({editing: true});
		return false;
	},
	onRun: function(e) {
		e.preventDefault();
		this.setState({running: true, error: ""});
		AdminAPI.runAnalyticsReport(this.props.reportID, function(success, data, error) {
			if (this.isMounted()) {
				this.updateResults(success, data, error);
			}
		}.bind(this));
		return false;
	},
	render: function() {
		// TODO: sandbox the iframe further by not allowing same-origin
		var form = null;
		if (this.state.editing) {
			form = (
				<div className="form">
					<form onSubmit={this.onSubmit}>
						<Forms.FormInput required type="text" label="Name" name="name" value={this.state.name} onChange={this.onNameChange} />
						<Gorms.TextArea tabs="true" label="Query" name="query" value={this.state.query} onChange={this.onQueryChange} rows="10" />
						<Gorms.TextArea tabs="true" label="Presentation" name="presentation" value={this.state.presentation} onChange={this.onPresentationChange} rows="15" />
						<div className="text-center">
							<button className="btn btn-default" onClick={this.onSave}>Save</button>
							&nbsp;<button disabled={this.state.results ? "" : "disabled"} className="btn btn-default" onClick={this.onDownload}>Download</button>
							&nbsp;<button type="submit" className="btn btn-primary">Query</button>
						</div>
					</form>
				</div>
			);
		}
		return (
			<div className="analytics">
				<div className="text-center">
					<h2>{this.state.name}</h2>
				</div>

				{this.state.editing ? form :
					<div className="form text-center">
						{Perms.has(Perms.AnalyticsReportsEdit) ? <button className="btn btn-default" onClick={this.onEdit}>Edit</button> : null}
						&nbsp;<button disabled={this.state.results ? "" : "disabled"} className="btn btn-default" onClick={this.onDownload}>Download</button>
						&nbsp;<button className="btn btn-primary" onClick={this.onRun}>Run</button>
					</div>}

				{this.state.error ? <Utils.Alert type="danger">{this.state.error}</Utils.Alert> : null}

				{this.state.running ? <Utils.Alert type="info">Querying... please wait</Utils.Alert> : null}

				{this.state.results && this.state.presentation ?
					<iframe sandbox="allow-scripts allow-same-origin" ref="presentation" src={"/admin/analytics/reports/"+this.props.reportID+"/presentation/iframe?v=" + this.state.version} border="0" width="100%" height="100%" />
					: null}

				{this.state.results ? AnalyticsTable({
					router: this.props.router,
					data: this.state.results
				}) : null}
			</div>
		);
	}
});

var AnalyticsTable = React.createClass({displayName: "AnalyticsTable",
	getInitialState: function() {
		return {sort: null, desc: false};
	},
	componentWillReceiveProps: function(nextProps) {
		this.setState(this.getInitialState());
	},
	onSort: function(col, e) {
		e.preventDefault();
		if (this.state.sort == col) {
			this.setState({desc: !this.state.desc});
		} else {
			this.setState({sort: col, desc: false});
		}
		return false;
	},
	render: function() {
		rows = this.props.data.rows || [];
		if (this.state.sort != null) {
			// Copy the rows before sorting to avoid mutating the original
			rows = rows.map(function(v) { return v; });
			rows.sort(function(a, b) {
				a = a[this.state.sort];
				b = b[this.state.sort];
				if (this.state.desc) {
					var t = a;
					a = b;
					b = t;
				}
				if (a < b) {
					return -1;
				}
				if (a > b) {
					return 1;
				}
				return 0;
			}.bind(this));
		}
		return (
			<div className="analytics-results">
				<div className="text-right">
					{rows.length} rows
				</div>
				<table className="table">
					<thead>
						<tr>
						{this.props.data.cols.map(function(col, index) {
							return (
								<th key={col}>
									<a href="#" onClick={this.onSort.bind(this, index)}>
										{col}
										{this.state.sort == index ?
											(this.state.desc ?
												<span className="glyphicon glyphicon-arrow-down" />
											:
												<span className="glyphicon glyphicon-arrow-up" />
											)
										:
											<span className="glyphicon">&nbsp;</span>
										}
									</a>
								</th>
							);
						}.bind(this))}
						</tr>
					</thead>
					<tfoot>
						<tr>
						{this.props.data.cols.map(function(col, index) {
							if (rows.length > 0 && typeof rows[0][index] == "number") {
								var sum = 0;
								rows.forEach(function(row) {
									sum += row[index];
								}.bind(this));
								return (
									<td key={"foot-"+col}>
										Sum: {sum}<br />
										Mean: {Math.round(100 * sum / rows.length) / 100}
									</td>
								);
							} else {
								return <td key={"foot-"+col}></td>;
							}
						}.bind(this))}
						</tr>
					</tfoot>
					<tbody>
						{rows.map(function(row, indexRow) {
							return (
								<tr key={"analytics-query-row-"+indexRow}>
									{row.map(function(v, indexVal) {
										return <td key={"analytics-query-row-"+indexRow+"-"+indexVal}>{v}</td>;
									}.bind(this))}
								</tr>
							);
						}.bind(this))}
					</tbody>
				</table>
			</div>
		)
	}
});
