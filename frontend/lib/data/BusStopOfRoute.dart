import 'dart:convert';

List<BusStopOfRoute> busStopOfRouteFromJson(dynamic str) {
  final List<dynamic> jsonData = str is String ? json.decode(str) : str;
  return List<BusStopOfRoute>.from(jsonData.map((x) => BusStopOfRoute.fromJson(x)));
}


class BusStop{
  final String StopUID;
  final String StopID;
  final Map<String,String> StopName;
  final int? StopBoarding;
  final int StopSequence;
  final double? PositionLon,PositionLat;
  final String? GeoHash;
  final String StationID;
  final String? StationGroupID;
  final String? LocationCityCode;
  BusStop(
    {
      required this.StopBoarding,
      required this.StopSequence,
      required this.StopUID,
      required this.StopID,
      required this.StopName,
      required this.PositionLon,
      required this.PositionLat,
      required this.GeoHash,
      required this.StationID,
      required this.LocationCityCode,
      required this.StationGroupID,
    }
  );
  factory BusStop.fromJson(Map<String,dynamic> stop){
    final Stopposition = stop["StopPosition"];
    return BusStop(
      StopUID: stop["StopUID"]?.toString() ?? "",
      StopID: stop["StopID"]?.toString() ?? "",
      StopName: stop["StopName"] is Map ? Map<String, String>.from(stop["StopName"]) : {},
      StopBoarding: stop["StopBoarding"],
      StopSequence: stop["StopSequence"],
      PositionLon: (Stopposition["PositionLon"] as num?)?.toDouble(),
      PositionLat: (Stopposition["PositionLat"] as num?)?.toDouble(),
      GeoHash: Stopposition["GeoHash"]?.toString(),
      StationID: stop["StationID"]?.toString() ?? "",
      LocationCityCode: stop["LocationCityCode"]?.toString(),
      StationGroupID: stop["StationGroupID"]?.toString(),
    );
  }
}

class BusStopOfRoute{
  final String RouteUID;
  final Map<String,String> RouteName;
  final String SubRouteUID;
  final Map<String,String> SubRouteName;
  final int Direction;
  final List<BusStop> Stops;
  final String UpdateTime;
  BusStopOfRoute(
    {
      required this.RouteUID,
      required this.RouteName,
      required this.SubRouteUID,
      required this.SubRouteName,
      required this.Direction,
      required this.Stops,
      required this.UpdateTime,
    }
  );
  factory BusStopOfRoute.fromJson(Map<String,dynamic> route){
    return BusStopOfRoute(
      RouteUID: route["RouteUID"]?.toString() ?? "",
      RouteName: route["RouteName"] is Map ? Map<String, String>.from(route["RouteName"]) : {},
      SubRouteUID: route["SubRouteUID"]?.toString() ?? "",
      SubRouteName: route["SubRouteName"] is Map ? Map<String, String>.from(route["SubRouteName"]) : {},
      Direction: route["Direction"] ?? 255,
      Stops: ((route["Stops"] as List?) ?? const [])
        .map((stop) => BusStop.fromJson(Map<String, dynamic>.from(stop)))
        .toList(),
      UpdateTime: route["UpdateTime"]?.toString() ?? "",
    );
  }
}