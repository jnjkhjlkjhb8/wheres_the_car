import 'dart:convert';

List<BusN1EstimateTime> busN1EstimateTimeFromJson(String str) => List<BusN1EstimateTime>.from(json.decode(str).map((x) => BusN1EstimateTime.fromJson(x)));

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
class BusN1EstimateTime{
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
  BusN1EstimateTime(
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
      required this.NextBusTime,
      required this.IsLastBus,
      required this.Estimates,
      required this.DataTime,
      required this.TransTime,
      required this.SrcRecTime,
      required this.SrcTransTime,
      required this.SrcUpdateTime,
      required this.UpdateTime
    }
  );
  factory BusN1EstimateTime.fromJson(Map<String,dynamic> busN1EstimateTime){
    return BusN1EstimateTime(
      PlateNumb: busN1EstimateTime["PlateNumb"] as String,
      StopUID: busN1EstimateTime["StopUID"] as String,
      StopID: busN1EstimateTime["StopID"] as String,
      StopName: Map<String,String>.from(busN1EstimateTime["StopName"]),
      RouteUID: busN1EstimateTime["RouteUID"] as String,
      RouteID: busN1EstimateTime["RouteID"] as String,
      RouteName: Map<String,String>.from(busN1EstimateTime["RouteName"]),
      SubRouteUID: busN1EstimateTime["SubRouteUID"] as String,
      SubRouteID: busN1EstimateTime["SubRouteID"] as String,
      SubRouteName: Map<String,String>.from(busN1EstimateTime["SubRouteName"]),
      Direction: busN1EstimateTime["Direction"] as int,
      EstimateTime: busN1EstimateTime["EstimateTime"] as int?,
      ScheduledTime: busN1EstimateTime["ScheduledTime"] as String?,
      StopCountDown: busN1EstimateTime["StopCountDown"] as int?,
      CurrentStop: busN1EstimateTime["CurrentStop"] as String?,
      DestinationStop: busN1EstimateTime["DestinationStop"] as String?,
      StopSequence: busN1EstimateTime["StopSequence"] as int,
      StopStatus: busN1EstimateTime["StopStatus"] as int,
      MessageType: busN1EstimateTime["MessageType"] as int?,
      NextBusTime: DateTime.parse(busN1EstimateTime["NextBusTime"]),
      IsLastBus: busN1EstimateTime["IsLastBus"] as bool?,
      Estimates: busN1EstimateTime["Estimates"] == null ? null : List<Estimate>.from(busN1EstimateTime["Estimates"].map((x) => Estimate.fromJson(x))),
      DataTime: busN1EstimateTime["DataTime"] == null ? null : DateTime.parse(busN1EstimateTime["DataTime"]),
      TransTime: busN1EstimateTime["TransTime"] == null ? null : DateTime.parse(busN1EstimateTime["TransTime"]),
      SrcRecTime: busN1EstimateTime["SrcRecTime"] == null ? null : DateTime.parse(busN1EstimateTime["SrcRecTime"]),
      SrcTransTime: busN1EstimateTime["SrcTransTime"] == null ? null : DateTime.parse(busN1EstimateTime["SrcTransTime"]),
      SrcUpdateTime: DateTime.parse(busN1EstimateTime["SrcUpdateTime"]),
      UpdateTime: DateTime.parse(busN1EstimateTime["UpdateTime"])
    );
  }
}