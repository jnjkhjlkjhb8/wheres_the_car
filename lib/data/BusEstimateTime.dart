import 'dart:convert';

List<BusEstimates> busEstimatesFromJson(dynamic str){
  final List<dynamic> jsonData = str is String ? json.decode(str) : str;
  return List<BusEstimates>.from(jsonData.map((x) => BusEstimates.fromJson(x)));
}

class Estimate{
  final String? PlateNumb;
  final int? EstimateTime;
  final bool? IsLastBus;
  final int? VehicleStopStatus;
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
      PlateNumb: estimate["PlateNumb"]?.toString() ?? "",
      EstimateTime: (estimate["EstimateTime"] as int?) ?? -1,
      IsLastBus: estimate["IsLastBus"] ?? false,  
      VehicleStopStatus: (estimate["VehicleStopStatus"] as int?) ?? -1
    );
  }
}
class BusEstimates{
  final String? PlateNumb;
  final String? StopUID;
  final String? StopID;
  final Map<String,String>? StopName;
  final String? RouteUID;
  final String? RouteID;
  final Map<String,String>? RouteName;
  final String? SubRouteUID;
  final String? SubRouteID;
  final Map<String,String>? SubRouteName;
  final int? Direction;
  final int? EstimateTime;
  final String? DestinationStop;
  final int? StopSequence;
  final int? StopStatus;
  final int? MessageType;
  final DateTime? NextBusTime;
  final bool? IsLastBus;
  final List<Estimate>? Estimates;
  final DateTime? UpdateTime;
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
      required this.DestinationStop,
      required this.StopSequence,
      required this.StopStatus,
      required this.MessageType,
      required this.IsLastBus,
      required this.Estimates,
      required this.UpdateTime,
      required this.NextBusTime
    }
  );
  factory BusEstimates.fromJson(Map<String,dynamic> estimate){
    return BusEstimates(
      PlateNumb: estimate["PlateNumb"] ?? "",
      StopUID: estimate["StopUID"] ?? "",
      StopID: estimate["StopID"] ?? "",
      StopName: Map<String,String>.from(estimate["StopName"] ?? {}),
      RouteUID: estimate["RouteUID"] ?? "",
      RouteID: estimate["RouteID"] ?? "",
      RouteName: Map<String,String>.from(estimate["RouteName"] ?? {}),
      SubRouteUID: estimate["SubRouteUID"] ?? "",
      SubRouteID: estimate["SubRouteID"] ?? "",
      SubRouteName: Map<String,String>.from(estimate["SubRouteName"] ?? {}),
      Direction: estimate["Direction"] ?? 255,
      EstimateTime: (estimate["EstimateTime"] as int?) ?? -1,
      DestinationStop: estimate["DestinationStop"] ?? "",
      StopSequence: (estimate["StopSequence"] as int?) ?? -1,
      StopStatus: (estimate["StopStatus"] as int?) ?? -1,
      MessageType: (estimate["MessageType"] as int?) ?? -1,
      IsLastBus: estimate["IsLastBus"] ?? false,
      Estimates: (estimate["Estimates"] as List?)
        ?.map((e) => Estimate.fromJson(e))
        .toList(),
      UpdateTime: DateTime.parse(estimate["UpdateTime"]),
      NextBusTime: DateTime.parse(estimate["NextBusTime"])
    );
  }
}