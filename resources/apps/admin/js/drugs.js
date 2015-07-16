/* @flow */

var AdminAPI = require("./api.js");
var React = require("react");
var Routing = require("../../libs/routing.js");
var Utils = require("../../libs/utils.js");

module.exports = {
	DrugSearch: React.createClass({displayName: "DrugSearch",
		mixins: [Routing.RouterNavigateMixin],
		getInitialState: function(): any {
			return {
				query: "",
				busy: false,
				error: null,
				results: null
			};
		},
		componentWillMount: function() {
			document.title = "Search | Drugs | Spruce Admin";
			var q = Utils.getParameterByName("q");
			if (q != this.state.query) {
				this.setState({query: q});
				this.search(q);
			}

			// TODO: need to make sure the page that was navigated to is this one
			// this.navCallback = (function() {
			// 	var q = getParameterByName("q");
			// 	if (q != this.state.query) {
			// 		this.setState({query: q});
			// 		this.search(q);
			// 	}
			// }).bind(this);
			// this.props.router.on("route", this.navCallback);
		},
		componentWillUnmount : function() {
			// this.props.router.off("route", this.navCallback);
		},
		search: function(q) {
			this.props.router.navigate("/carecoordinator/drugs?q=" + encodeURIComponent(q), {replace: true}); // TODO: replacing until back tracking works
			if (q == "") {
				this.setState({results: null});
			} else {
				this.setState({busy: true, error: null});
				AdminAPI.searchDrugs(q, function(success, res, error) {
					if (this.isMounted()) {
						if (!success) {
							this.setState({busy: false, results: null, defailts: {}, error: error.message});
							return;
						}
						this.setState({
							busy: false,
							error: null,
							results: res.results || []
						});
					}
				}.bind(this));
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
				<div>
					<div>
						<div style={{maxWidth: 400, margin: "0 auto", textAlign: "center"}} >
							<h2>Search drugs</h2>
							<form onSubmit={this.onSearchSubmit}>
								<div className="form-group">
									<input required autofocus
										type = "text"
										className = "form-control"
										name = "q"
										value = {this.state.query}
										onChange = {this.onQueryChange} />
								</div>
								<button type="submit" className="btn btn-primary btn-lg center-block">Search</button>
							</form>
						</div>
					</div>

					<div className="search-results">
						{this.state.busy ?
							<div className="text-center"><Utils.LoadingAnimation /></div>
						:
							<div>
							{this.state.error ? <div className="text-center"><Utils.Alert type="danger">{this.state.error}</Utils.Alert></div> : null}

							{this.state.results ?
								<DrugSearchResults
									router = {this.props.router}
									results = {this.state.results} />
							: null}
							</div>
						}
					</div>
				</div>
			);
		}
	})
};

var DrugSearchResults = React.createClass({displayName: "DrugSearchResults",
	mixins: [Routing.RouterNavigateMixin],
	render: function(): any {
		if (this.props.results.length == 0) {
			return (<div className="no-results text-center">No matching drugs found</div>);
		}

		var results = this.props.results.map(function (res) {
			return (
				<div key={res.name}>
					<DrugSearchResult result={res} router={this.props.router} />
				</div>
			);
		}.bind(this))

		return (
			<div>{results}</div>
		);
	}
});

var DrugSearchResult = React.createClass({displayName: "DrugSearchResult",
	mixins: [Routing.RouterNavigateMixin],
	render: function(): any {
		return (
			<div>
				<h3>{this.props.result.name}</h3>
				{this.props.result.error ?
					<div><strong>ERROR: {this.props.result.error}</strong></div>
				:
					<table className="table">
					<thead>
						<tr>
							<th>Strength</th>
							<th>Generic Product Name</th>
							<th>Parsed Generic Name</th>
							<th>Route</th>
							<th>Form</th>
							<th>Dispense Unit</th>
							<th>Sched</th>
							<th>OTC</th>
							<th>NDC</th>
							<th>Guide</th>
						</tr>
					</thead>
					<tbody>
					{this.props.result.strengths.map(function(st) {
						var ndc = st.medication.RepresentativeNDC;
						if (st.error) {
							return <tr key={st.strength}><td><strong>ERROR: {st.error}</strong></td></tr>
						}
						return (
							<tr key={st.medication.GenericProductName}>
								<td>{st.strength}</td>
								<td>{st.medication.GenericProductName}</td>
								<td>{st.parsed_generic_name}</td>
								<td>{st.medication.RouteDescription}</td>
								<td>{st.medication.DoseFormDescription}</td>
								<td>{st.medication.DispenseUnitDescription}</td>
								<td>{st.medication.Schedule}</td>
								<td>{st.medication.OTC ? "true" : "false"}</td>
								<td>{ndc}</td>
								<td>
									{st.guide_id && st.guide_id != "0" ?
										<a href={"/admin/guides/rx/" + st.guide_id} onClick={this.onNavigate}>Guide</a> : null}
								</td>
							</tr>
						);
					}.bind(this))}
					</tbody>
					</table>
				}
			</div>
		);
	}
});
