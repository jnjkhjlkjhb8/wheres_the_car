import 'dart:convert';

List<BusStopOfRoute> BusStopOfRouteFromJson(String str) => List<BusStopOfRoute>.from(json.decode(str).map((x) => BusStopOfRoute.fromJson(x)));

class BusOperators{
  final String operatorID;
  final Map<String,String> operatorName;
  final String operatorCode;
  final String OperatorNo;
  BusOperators(
    {
      required this.operatorID,
      required this.operatorName,
      required this.operatorCode,
      required this.OperatorNo
    }
  );
  factory BusOperators.fromJson(Map<String,dynamic> operator){
    return BusOperators(
      operatorID: operator["OperatorID"],
      operatorName: Map<String,String>.from(operator["OperatorName"]),
      operatorCode: operator["OperatorCode"],
      OperatorNo: operator["OperatorNo"]
    );
  }
}
class BusStop{
  final String StopUID;
  final String StopID;
  final Map<String,String> StopName;
  final int StopBoarding;
  final int StopSequence;
  final double PositionLon,PositionLat;
  final String GeoHash;
  final String StationID;
  final String LocationCityCode;
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
      required this.LocationCityCode
    }
  );
  factory BusStop.fromJson(Map<String,dynamic> stop){
    final Stopposition = stop["StopPosition"] as Map<String, dynamic>;
    return BusStop(
      StopUID: stop["StopUID"],
      StopID: stop["StopID"],
      StopName: Map<String,String>.from(stop["StopName"]),
      StopBoarding: stop["StopBoarding"],
      StopSequence: stop["StopSequence"],
      PositionLon: Stopposition["PositionLon"],
      PositionLat: Stopposition["PositionLat"],
      GeoHash: Stopposition["GeoHash"],
      StationID: stop["StationID"],
      LocationCityCode: stop["LocationCityCode"]
    );
  }
}

class BusStopOfRoute{
  final String RouteUID;
  final String RouteID;
  final Map<String,String> RouteName;
  final List<BusOperators> Operators;
  final String SubRouteUID;
  final String SubRouteID;
  final Map<String,String> SubRouteName;
  final int Direction;
  final String City;
  final String CityCode;
  final List<BusStop> Stops;
  final String UpdateTime;
  final int VersionID;
  BusStopOfRoute(
    {
      required this.RouteUID,
      required this.RouteID,
      required this.RouteName,
      required this.SubRouteUID,
      required this.SubRouteID,   
      required this.SubRouteName,
      required this.Direction,
      required this.City,
      required this.CityCode,
      required this.Stops,
      required this.Operators,
      required this.UpdateTime,
      required this.VersionID
    }
  );
  factory BusStopOfRoute.fromJson(Map<String,dynamic> route){
    return BusStopOfRoute(
      RouteUID: route["RouteUID"],
      RouteID: route["RouteID"],
      RouteName: Map<String,String>.from(route["RouteName"]),
      Operators: (route["Operators"] as List)
        .map((operator) => BusOperators.fromJson(operator))
        .toList(),
      SubRouteUID: route["SubRouteUID"],
      SubRouteID: route["SubRouteID"],
      SubRouteName: Map<String,String>.from(route["SubRouteName"]),
      Direction: route["Direction"],
      City: route["City"],
      CityCode: route["CityCode"],
      Stops: (route["Stops"] as List)
        .map((stop) => BusStop.fromJson(stop))
        .toList(),
      UpdateTime: route["UpdateTime"],
      VersionID: route["VersionID"]
    );
  }
}