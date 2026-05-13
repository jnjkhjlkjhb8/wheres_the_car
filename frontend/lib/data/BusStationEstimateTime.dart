import 'dart:convert';

List<BusN1EstimateTime> busN1EstimateTimeFromJson(dynamic str) {
  final List<dynamic> jsonData = str is String ? json.decode(str) : str;
  return List<BusN1EstimateTime>.from(jsonData.map((x) => BusN1EstimateTime.fromJson(x)));
}

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
      PlateNumb: estimate["PlateNumb"]?.toString() ?? "",
      EstimateTime: estimate["EstimateTime"] as int? ?? -1,
      IsLastBus: estimate["IsLastBus"] as bool? ?? false,
      VehicleStopStatus: estimate["VehicleStopStatus"] as int? ?? -1
    );
  }
}
class BusN1EstimateTime{
  final String PlateNumb;
  final String SubRouteUID;
  final Map<String,String> SubRouteName;
  final int? EstimateTime;
  final String? ScheduledTime;
  final String? DestinationStop;
  final int StopStatus;
  final DateTime? NextBusTime;
  final bool? IsLastBus;
  final List<Estimate>? Estimates;
  final DateTime SrcUpdateTime;
  final DateTime UpdateTime;
  BusN1EstimateTime(
    {
      required this.PlateNumb,
      required this.SubRouteUID,
      required this.SubRouteName,
      required this.EstimateTime,
      required this.ScheduledTime,
      required this.DestinationStop,
      required this.StopStatus,
      required this.NextBusTime,
      required this.IsLastBus,
      required this.Estimates,
      required this.UpdateTime,
      required this.SrcUpdateTime
    }
  );
  factory BusN1EstimateTime.fromJson(Map<String,dynamic> busN1EstimateTime){
    return BusN1EstimateTime(
      PlateNumb: busN1EstimateTime["PlateNumb"] as String,
      SubRouteUID: busN1EstimateTime["SubRouteUID"] as String,
      SubRouteName: Map<String,String>.from(busN1EstimateTime["SubRouteName"]),
      EstimateTime: busN1EstimateTime["EstimateTime"] as int?,
      ScheduledTime: busN1EstimateTime["ScheduledTime"] as String?,
      DestinationStop: busN1EstimateTime["DestinationStop"] as String?,
      StopStatus: busN1EstimateTime["StopStatus"] as int,
      NextBusTime: busN1EstimateTime["NextBusTime"] == null ? null : DateTime.parse(busN1EstimateTime["NextBusTime"]),
      IsLastBus: busN1EstimateTime["IsLastBus"] as bool?,
      Estimates: busN1EstimateTime["Estimates"] == null ? null : List<Estimate>.from(busN1EstimateTime["Estimates"].map((x) => Estimate.fromJson(x))),
      UpdateTime: DateTime.parse(busN1EstimateTime["UpdateTime"]),
      SrcUpdateTime: DateTime.parse(busN1EstimateTime["SrcUpdateTime"])
    );
  }
}