import 'dart:convert';

List<BusEstimates> busEstimatesFromJson(String str) => List<BusEstimates>.from(json.decode(str).map((x) => BusEstimates.fromJson(x)));

class Estimate{
  final String PlateNumb;
  final int EstimateTime;
  final bool IsLastBus;
  final int VehicleStopStatus;
  Estimate(
    {
      required this.PlateNumb,
      required this.EstimateTime,
      required this.IsLastBus,
      required this.VehicleStopStatus
    }
  );
  factory Estimate.fromJson(Map<String,dynamic> estimate){
    return Estimate(
      PlateNumb: estimate["PlateNumb"] as String,
      EstimateTime: estimate["EstimateTime"] as int,
      IsLastBus: estimate["IsLastBus"] as bool,
      VehicleStopStatus: estimate["VehicleStopStatus"] as int
    );
  }
}
class BusEstimates{
  final String PlateNumb;
  final String StopUID;
  final String StopID;
  final Map<String,String> StopName;
  final String RouteUID;
  final String RouteID;
  final Map<String,String> RouteName;
  final String SubRouteUID;
  final String SubRouteID;
  final Map<String,String> SubRouteName;
  final int Direction;
  final int? EstimateTime;
  final String? ScheduledTime;
  final int? StopCountDown;
  final String? CurrentStop;
  final String? DestinationStop;
  final int StopSequence;
  final int StopStatus;
  final int? MessageType;
  final DateTime NextBusTime;
  final bool? IsLastBus;
  final List<Estimate>? Estimates;
  final DateTime? DataTime;
  final DateTime? TransTime;
  final DateTime? SrcRecTime;
  final DateTime? SrcTransTime;
  final DateTime SrcUpdateTime;
  final DateTime UpdateTime;
  BusEstimates(
    {
      required this.PlateNumb,
      required this.StopUID,
      required this.StopID,
      required this.StopName,
      required this.RouteUID,
      required this.RouteID,
      required this.RouteName,
      required this.SubRouteUID,
      required this.SubRouteID,
      required this.SubRouteName,
      required this.Direction,
      required this.EstimateTime,
      required this.ScheduledTime,
      required this.StopCountDown,
      required this.CurrentStop,
      required this.DestinationStop,
      required this.StopSequence,
      required this.StopStatus,
      required this.MessageType,
      required this.IsLastBus,
      required this.Estimates,
      required this.DataTime,
      required this.TransTime,
      required this.SrcRecTime,
      required this.SrcTransTime,
      required this.SrcUpdateTime,
      required this.UpdateTime,
      required this.NextBusTime
    }
  );
  factory BusEstimates.fromJson(Map<String,dynamic> estimate){
    return BusEstimates(
      PlateNumb: estimate["PlateNumb"],
      StopUID: estimate["StopUID"],
      StopID: estimate["StopID"],
      StopName: Map<String,String>.from(estimate["StopName"]),
      RouteUID: estimate["RouteUID"],
      RouteID: estimate["RouteID"],
      RouteName: Map<String,String>.from(estimate["RouteName"]),
      SubRouteUID: estimate["SubRouteUID"],
      SubRouteID: estimate["SubRouteID"],
      SubRouteName: Map<String,String>.from(estimate["SubRouteName"]),
      Direction: estimate["Direction"],
      EstimateTime: estimate["EstimateTime"],
      ScheduledTime: estimate["ScheduledTime"],
      StopCountDown: estimate["StopCountDown"],
      CurrentStop: estimate["CurrentStop"],
      DestinationStop: estimate["DestinationStop"],
      StopSequence: estimate["StopSequence"],
      StopStatus: estimate["StopStatus"],
      MessageType: estimate["MessageType"],
      IsLastBus: estimate["IsLastBus"],
      Estimates: (estimate["Estimates"] as List?)
        ?.map((e) => Estimate.fromJson(e))
        .toList(),
      DataTime: DateTime.parse(estimate["DataTime"]),
      TransTime: DateTime.parse(estimate["TransTime"]),
      SrcRecTime: DateTime.parse(estimate["SrcRecTime"]),
      SrcTransTime: DateTime.parse(estimate["SrcTransTime"]),
      SrcUpdateTime: DateTime.parse(estimate["SrcUpdateTime"]),
      UpdateTime: DateTime.parse(estimate["UpdateTime"]),
      NextBusTime: DateTime.parse(estimate["NextBusTime"])
    );
  }
}