/** @jsx React.DOM */

var AdminAPI = require("../admin/api.js");
var Utils = require("../utils.js");

window.Dashboard = React.createClass({displayName: "Dashboard",
	getDefaultProps: function() {
		return {};
	},
	componentDidMount: function() {
		$(this.getDOMNode()).dblclick(function() {
			console.log("BOOM");
			Utils.fullscreen(document.documentElement);
		});
	},
	render: function() {
		return (
			<div>
				<div className="col-lg-3 col-md-4 col-sm-6 widget-container">
					<AnalyticsReportWidget reportID="2" type="line" />
				</div>
				<div className="col-lg-3 col-md-4 col-sm-6 widget-container">
					<LibratoCompositeWidget title="REST API Success Rate (last hour)"
						query='divide([derive(sum(s("restapi.requests.response.500", "*")), {detect_reset:"true"}), derive(sum(s("restapi.requests.response.*", "*")), {detect_reset:"true"})])'
						transform={function(v) { return 100 * (1-v); }} />
				</div>
				<div className="col-lg-3 col-md-4 col-sm-6 widget-container">
					<AnalyticsReportWidget reportID="4" type="number" />
				</div>
				<div className="col-lg-3 col-md-4 col-sm-6 widget-container">
					<StripeChargesWidget title="Stripe Charges" />
				</div>
				<div className="col-lg-6 col-md-8 col-sm-12 widget-container">
					<LibratoCompositeWidget title="REST API Requests (last hour)" query='sum(s("restapi.requests.total", "prod*", {function: "sum"}))' />
				</div>
				<div className="col-lg-6 col-md-8 col-sm-12 widget-container">
					<LibratoCompositeWidget title="REST API Requests (last hour)" query='sum(s("restapi.requests.total", "prod*", {function: "sum"}))' />
				</div>
			</div>
		);
	}
});

AnalyticsReportWidget = React.createClass({displayName: "AnalyticsReportWidget",
	componentWillMount: function() {
		this.loadReport(this.props.reportID);
	},
	componentWillReceiveProps: function(nextProps) {
		if (this.props.reportID != nextProps.reportID) {
			this.loadReport(nextProps.reportID);
		}
	},
	getInitialState: function() {
		return {running: false, error: "", results: {cols: null, rows: null}};
	},
	loadReport: function(id) {
		this.setState({running: true});
		AdminAPI.analyticsReport(id, function(success, report, error) {
			if (this.isMounted()) {
				if (!success) {
					this.setState({running: false, error: "ERROR: " + error.message});
					return
				}
				this.setState({
					name: report.name,
					error: ""
				});
				this.runReport(id);
			}
		}.bind(this));
	},
	runReport: function(id) {
		this.setState({running: true});
		AdminAPI.runAnalyticsReport(id, function(success, res, error) {
			if (this.isMounted()) {
				if (!success) {
					this.setState({running: false, error: "ERROR: " + error.message});
					return;
				}
				if (res.error) {
					this.setState({
						running: false,
						error: res.error
					});
					return;
				}
				this.setState({
					running: false,
					error: "",
					results: {
						cols: res.cols,
						rows: res.rows
					}
				});
			}
		}.bind(this));
	},
	render: function() {
		var body = "";
		if (this.state.running) {
			body = <Utils.LoadingAnimation />;
		} else if (this.state.error) {
			body = this.state.error;
		} else {
			if (this.props.type == "line") {
				body = <LineChart data={this.state.results.rows} />;
			} else if (this.props.type = "number") {
				var num = "?";
				if (this.state.results.rows.length > 0) {
					num = this.state.results.rows[0];
				}
				body = <div className="text-center bignum">{num}</div>;
			} else {
				body = "Unknown type: " + this.props.type;
			}
		}
		return (
			<div className="widget">
				<div className="title">{this.state.name}</div>
				<div className="body">{body}</div>
			</div>
		);
	}
});

LineChart = React.createClass({displayName: "LineChart",
	getInitialState: function() {
		return {created: false};
	},
	componentDidMount: function() {
		if (this.props.data)  {
			this.renderGraph(this.props.data);
		}
	},
	componentWillReceiveProps: function(newProps) {
		if (newProps.data) {
			this.renderGraph(newProps.data);
		}
	},
	renderGraph: function(data) {
		if (!data || data.length == 0) {
			return;
		}

		var dateFormat = d3.time.format("%Y-%m-%d");
		var startDate = dateFormat.parse(data[0][0]);
		var endDate = dateFormat.parse(data[data.length-1][0]);
		if (endDate < startDate) {
			var t = startDate;
			startDate = endDate;
			endDate = t;
		}
		endDate = d3.time.day.offset(endDate, 1);
		var days = d3.time.days(startDate, endDate).map(dateFormat);

		var valueMap = {};
		for(var i = 0; i < data.length; i++) {
			valueMap[data[i][0]] = data[i][1];
		}
		var values = days.map(function(d) { return valueMap[d] || 0; });

		var data = {
		    labels: days.map(function(r) { return ""; }),
		    datasets: [
		        {
		            fillColor: "rgba(151,187,205,0.2)",
		            strokeColor: "rgba(151,187,205,1)",
		            pointColor: "rgba(151,187,205,1)",
		            pointStrokeColor: "#fff",
		            pointHighlightFill: "#fff",
		            pointHighlightStroke: "rgba(151,187,205,1)",
		            data: values
		        }
		    ]
		};
		var options = {
		    pointDot : false,
		    bezierCurveTension : 0.1
		};

		var canvas = this.getDOMNode();
		canvas.width = canvas.parentNode.clientWidth - 20;

		var ctx = canvas.getContext("2d");
		var chart = new Chart(ctx).Line(data, options);
	},
	render: function() {
		return (
			<canvas height="190"></canvas>
		);
	}
});

LineChart2 = React.createClass({displayName: "LineChart2",
	getInitialState: function() {
		return {created: false};
	},
	resizeCanvas: function() {
		this.renderGraph(this.props.data);
	},
	componentDidMount: function() {
		// $(window).bind("resize", this.resizeCanvas);
		if (this.props.data)  {
			this.renderGraph(this.props.data);
		}
	},
	componentWillUnmount: function() {
		// $(window).unbind("resize", this.resizeCanvas);
	},
	componentWillReceiveProps: function(newProps) {
		if (newProps.data) {
			this.renderGraph(newProps.data);
		}
	},
	renderGraph: function(data) {
		if (!data || data.length == 0) {
			return;
		}

		var data = {
		    labels: data.map(function(r) { return ""; /*r[0]*/ }),
		    datasets: [
		        {
		            fillColor: "rgba(151,187,205,0.2)",
		            strokeColor: "rgba(151,187,205,1)",
		            pointColor: "rgba(151,187,205,1)",
		            pointStrokeColor: "#fff",
		            pointHighlightFill: "#fff",
		            pointHighlightStroke: "rgba(151,187,205,1)",
		            data: data.map(function(r) { return r[1]; })
		        }
		    ]
		};
		var options = {
		    pointDot: false,
		    bezierCurveTension: 0.1
		};

		var canvas = this.getDOMNode();
		canvas.width = canvas.parentNode.clientWidth - 20;

		var ctx = canvas.getContext("2d");
		var chart = new Chart(ctx).Line(data, options);
	},
	render: function() {
		return (
			<canvas height="190"></canvas>
		);
	}
});

LibratoCompositeWidget = React.createClass({displayName: "LibratoCompositeWidget",
	getDefaultProps: function() {
		return {transform: function(x) { return x; }};
	},
	getInitialState: function() {
		return {data: null};
	},
	componentWillMount: function() {
		var startTime = Math.round(new Date().getTime() / 1000 - 60*60);
		AdminAPI.libratoQueryComposite(this.props.query, 60, startTime, 0, 0, function(success, data, error) {
			var rows = data.measurements[0].series.map(function(row) {
				var d = Utils.unixTimestampToDate(row.measure_time);
				return [d3.time.format("%Y-%m-%d %X")(d), this.props.transform(row.value)];
			}.bind(this));
			this.setState({data: rows});
		}.bind(this));
	},
	render: function() {
		return (
			<div className="widget">
				<div className="title">{this.props.title}</div>
				<div className="body"><LineChart2 data={this.state.data} /></div>
			</div>
		);
	}
});

StripeChargesWidget = React.createClass({displayName: "StripeChargesWidget",
	getInitialState: function() {
		return {busy: false, data: null, error: ""};
	},
	componentWillMount: function() {
		this.setState({busy: true});
		AdminAPI.stripeCharges(40, function(success, data, error) {
			if (this.isMounted()) {
				if (!success) {
					this.setState({busy: false, error: error});
					return;
				}
				var days = {};
				var dates = data.map(function(d) {
					var day = d.created.substring(0, 10);
					days[day] = (days[day] || 0) + d.amount;
					return day;
				});
				var seen = {};
				var dayTotals = [];
				dates.forEach(function(day) {
					if (seen[day]) {
						return;
					}
					dayTotals.push([day, days[day] / 100]);
					seen[day] = true;
				});
				for(var i = 0; i < Math.floor(dayTotals.length/2); i++) {
					var t = dayTotals[i];
					dayTotals[i] = dayTotals[dayTotals.length-i-1];
					dayTotals[dayTotals.length-i-1] = t;
				}
				this.setState({data: dayTotals, busy: false, error: ""});
			}
		}.bind(this));
	},
	render: function() {
		return (
			<div className="widget">
				<div className="title">{this.props.title}</div>
				<div className="body"><LineChart data={this.state.data} /></div>
			</div>
		);
	}
});
