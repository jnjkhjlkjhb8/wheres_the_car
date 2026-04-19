import 'dart:convert';

List<BusShape> BusShapeFromJson(String str) => List<BusShape>.from(json.decode(str).map((x) => BusShape.fromJson(x)));

class BusShape{
  final String RouteUID;
  final String RouteID;
  final Map<String,String> RouteName;
  final String SubRouteUID;
  final String SubRouteID;
  final Map<String,String> SubRouteName;
  final int Direction;
  final String Geometry;
  final String EncodedPolyline;
  final DateTime UpdateTime;
  final int VersionID;
  BusShape(
    {
      required this.RouteUID,
      required this.RouteID,
      required this.RouteName,
      required this.SubRouteUID,
      required this.SubRouteID,
      required this.SubRouteName,
      required this.Direction,
      required this.Geometry,
      required this.EncodedPolyline,
      required this.UpdateTime,
      required this.VersionID
    }
  );
  factory BusShape.fromJson(Map<String,dynamic> subroute){
    return BusShape(
      RouteUID: subroute["RouteUID"],
      RouteID: subroute["RouteID"],
      RouteName: Map<String,String>.from(subroute["RouteName"]),
      SubRouteUID: subroute["SubRouteUID"],
      SubRouteID: subroute["SubRouteID"],
      SubRouteName: Map<String,String>.from(subroute["SubRouteName"]),
      Direction: subroute["Direction"],
      Geometry: subroute["Geometry"],
      EncodedPolyline: subroute["EncodedPolyline"],
      UpdateTime: DateTime.parse(subroute["UpdateTime"]),
      VersionID: subroute["VersionID"]
    );
  }
}