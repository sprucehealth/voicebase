/* @flow */

var AdminAPI = require("./api.js");
var Nav = require("../../libs/nav.js");
var Utils = require("../../libs/utils.js");
var React = require("react");
var Routing = require("../../libs/routing.js");
var Forms = require("../../libs/forms.js");

module.exports = {
	Page: React.createClass({
		mixins: [Routing.RouterNavigateMixin],
		menuItems: function(): LeftNavMenu {
			var items = [
			{
				id: "incoming",
				url: "/admin/financial/incoming",
				name: "Incoming Items"
			},
			{
				id: "outgoing",
				url: "/admin/financial/outgoing",
				name: "Outgoing Items"
			}];
			return [items];
		},
		getDefaultProps: function(): any {
			return {}
		},
		pages: {
			outgoing: function(): any {
				return <QueryableItems
					title = "Outgoing Items"
					description = "Outgoing items represent items that result in a pay out. For example, a doctor completing the first treatment plan for a case generates an outgoing item."
					documentTitle = "Financial | Outgoing Items | Spruce Admin"
					path = "financial/outgoing"
					headerFields = {["Created (UTC)", "Type", "Receipt ID", "Item ID", "State", "Doctor Name"]}
					id = "outgoing"
					fetchItems = {AdminAPI.outgoingFinancialItems.bind(AdminAPI)}
					resultKeys = {[
						{
							name: "Created",
							clickable: false
						},
						{
							name: "SKUType",
							clickable: false
						},
						{
							name: "ReceiptID",
							clickable: false
						},
						{
							name: "ItemID",
							clickable: false
						},
						{
							name: "State",
							clickable: false
						},
						{
							name: "Name",
							clickable: false
						}]}
						sortKey = {"Created"}
						router={this.props.router} />;
			},
			incoming: function(): any {
				var paymentsLink = "https://dashboard.stripe.com/test/payments/<value>"
				if (Spruce.Environment == "prod") {
					paymentsLink = "https://dashboard.stripe.com/payments/<value>"
				}
				return <QueryableItems
						router = {this.props.router}
						title = "Incoming Items"
						description = "Incoming items items that were charged for on Spruce. For example, a patient being charged for a visit submission (even if the charge is $0) generates an incoming item."
						documentTitle = "Financial | Incoming Items | Spruce Admin"
						path = "financial/incoming"
						id = "incoming"
						headerFields = {["Created (UTC)", "Charge ID", "Type", "Receipt ID", "Item ID", "State"]}
						fetchItems = {AdminAPI.incomingFinancialItems.bind(AdminAPI)}
						resultKeys = {[
							{
								name: "Created",
								clickable: false
							},
							{
								name: "ChargeID",
								clickable: true,
								link: paymentsLink
							},
							{
								name: "SKUType",
								clickable: false
							},
							{
								name: "ReceiptID",
								clickable: false
							},
							{
								name: "ItemID",
								clickable: false
							},
							{
								name: "State",
								clickable: false
							}]}
						sortKey = {"Created"}
						/>;
			},
		},
		render: function(): any {
			return (
				<div>
					<Nav.LeftNav router={this.props.router} items={this.menuItems()} currentPage={this.props.page}>
						{this.pages[this.props.page].bind(this)()}
					</Nav.LeftNav>
				</div>
			);
		}
	}),
	AvailableVisitSKUsSelect : React.createClass({displayName: "AvailableVisitSKUsSelect",
		mixins: [Routing.RouterNavigateMixin],
		getInitialState: function(): any {
			return {
				skus: [],
				busy: false,
				error: null,
				selectedValue: "",
			};
		},
		componentWillMount: function() {
			this.setState({busy: true});
			AdminAPI.visitSKUs(true, function(success, data, error) {
				if (this.isMounted()) {
					if (success) {
						var skus = data.skus.map(function(s) { return {name: s, value: s} });
						this.setState({
							busy: false,
							error: null,
							skus: skus,
							selectedValue: skus.length == 0  ? "" : skus[0].value
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
		onChange: function(e: any) {
			this.props.onChange(e, e.target.value, e.target.value)
			this.setState({
				selectedValue: e.target.value
			});
		},
		render: function(): any {
			return (
				<div className="skus-select">
					<form>
						<Forms.FormSelect onChange={this.onChange} value={this.state.selectedValue} opts={this.state.skus} />
					</form>
				</div>
			);
		}
	})
};

function DownloadAsCSV(headerFields, resultKeys, results, name) {
	// Generate CSV of the results
	var csv = headerFields.join(",");

	for (var i = 0; i < results.length; i++) {
		var row = results[i];
		csv += "\n" + resultKeys.map(function(resultKey) {
			var col = row[resultKey.name] || ""
			if (typeof col == "number") {
				return col;
			} else if (typeof col == "string") {
				return '"' + col.replace(/"/g, '""') + '"';
			} else  {
				return '"' + col.toString().replace(/"/g, '""') + '"';
			}
		}).join(",");
	}

	name = name || "report";

	var pom = document.createElement('a');
	pom.setAttribute('href', 'data:application/octet-binary;charset=utf-8,' + encodeURIComponent(csv));
	pom.setAttribute('download', name + ".csv");
	pom.click();
}

var QueryableItems = React.createClass({
	mixins: [Routing.RouterNavigateMixin],
	getInitialState: function(): any {
		return {
			error: null,
			busy: false,
			resultItems: [],
			from: "",
			to: ""
		};
	},
	componentWillMount: function() {
		document.title = this.props.documentTitle;
		var fromDate = Utils.getParameterByName("from");
		var toDate = Utils.getParameterByName("to");
		if (fromDate != this.state.from || toDate != this.state.to) {
			this.setState({from: fromDate, to: toDate, error: null});
			this.fetchItems(fromDate, toDate);
		}
	},
	componentWillReceiveProps: function(nextProps) {
		// force update the state when props change
		this.setState({from:"", to:"", resultItems:[], error:null, busy:false});
	},
	fetchItems: function(fromDate, toDate) {
		this.setState({busy: true, error: null, resultItems: []})
		this.props.fetchItems(fromDate, toDate, function(success, res, error) {
			if (this.isMounted()) {
				if (success) {
					this.sortResults(res.items);
					this.setState({busy: false, resultItems: res.items || [] })
				} else {
					this.setState({busy: false, error: error.message})
				}
			}
		}.bind(this));
	},
	sortResults: function(results) {
		results.sort(function(a, b) {
			if (a[this.props.sortKey] > b[this.props.sortKey]) {
				return 1;
			} else if (a[this.props.sortKey] < b[this.props.sortKey]) {
				return -1;
			}
			return 0;
		}.bind(this));
	},
	onQuery: function() {
		this.props.router.navigate(this.props.path+"?from="+encodeURIComponent(this.state.from)+"&to="+encodeURIComponent(this.state.to), {replace: true});
		this.fetchItems(this.state.from, this.state.to);
	},
	onUserInput: function(fromTime, toTime) {
		this.setState({from: fromTime, to: toTime});
	},
	exportResults: function() {
		var fileName = this.props.id+"-"+this.state.from+"-"+this.state.to;
		DownloadAsCSV(this.props.headerFields, this.props.resultKeys, this.state.resultItems, fileName);
	},
	render: function(): any {
		return (
			<div>

				<h2>{this.props.title}</h2>
				<p>{this.props.description}</p>

				<div style={{marginTop: 30}}>
					<QueryBar
						onQuery={this.onQuery}
						onUserInput={this.onUserInput}
						from={this.state.from}
						to={this.state.to}
					/>

					{this.state.resultItems.length > 0 ?
						<div className="pull-right">
							<br/>
							<button className="btn btn-default" onClick={this.exportResults}>Export</button>
						</div>
						: null}

					<ResultsContainer headerFields={this.props.headerFields} resultKeys={this.props.resultKeys} resultRows={this.state.resultItems}/>
					<div className="text-center">
						{this.state.busy ? <Utils.LoadingAnimation /> : null}
						{this.state.error ? <Utils.Alert type="danger">{this.state.error}</Utils.Alert> : null}
					</div>
				</div>
			</div>
		);
	}
});

var ResultsContainer = React.createClass({
	render: function(): any {

		var headerElements = this.props.headerFields.map(function(hf, i) {
			return <th key={"header-"+i}>{hf}</th>;
		});

		var rows = [];
		if (this.props.resultRows.length > 0) {
			this.props.resultRows.forEach(function(resultRow, rowIndex){

				var cols = [];
				this.props.resultKeys.forEach(function(key, colIndex){
					if (key.clickable) {
						var value = resultRow[key.name];
						var link = key.link.replace("<value>", value);
						cols.push(
							<td key={"element-"+colIndex}>
								<a href={link}>{resultRow[key.name]}</a>
							</td>
						);
					} else {
						cols.push(<td key={"element-"+colIndex}>{resultRow[key.name]}</td>)
					}
				}.bind(this));

				rows.push(
					<tr key={"item-"+rowIndex}>
						{cols}
					</tr>
				);

			}.bind(this));
		}


		return (
			<table className="table">
				<thead>
					<tr>
						{headerElements}
					</tr>
				</thead>
				<tbody>
					{rows}
				</tbody>
			</table>
		);
	}
});

var QueryBar = React.createClass({displayName: "QueryBar",
	handleChange: function() {
		this.props.onUserInput(
			this.refs.fromDateInput.getDOMNode().value,
			this.refs.toDateInput.getDOMNode().value
		);
	},
	handleQuery: function(e) {
		e.preventDefault();
		this.props.onQuery();
	},
	render: function(): any {
		return (
			<div>
				<div className="col-md-3">
					<strong>From</strong>
					<input required
						type="date"
						id="From"
						ref="fromDateInput"
						value={this.props.from}
						placeholder="MM/DD/YYYY"
						className="form-control"
						onChange={this.handleChange}/>
				</div>

				<div className="col-md-3">
					<strong>To</strong>
					<input required
						type="date"
						id="To"
						ref="toDateInput"
						value={this.props.to}
						placeholder="MM/DD/YYYY"
						className="form-control"
						onChange={this.handleChange}/>
				</div>

				<div className="col-md-3">
					<br/>
					<button type="submit" className="btn btn-primary" onClick={this.handleQuery}>Query</button>
				</div>
			</div>
		);
	}
});

