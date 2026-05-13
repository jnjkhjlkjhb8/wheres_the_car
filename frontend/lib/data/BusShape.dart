import 'dart:convert';

List<BusShape> BusShapeFromJson(String str) => List<BusShape>.from(json.decode(str).map((x) => BusShape.fromJson(x)));

class BusShape{
  final String SubRouteUID;
  final Map<String,String> SubRouteName;
  final int? Direction;
  final String EncodedPolyline;
  final DateTime UpdateTime;
  BusShape(
    {
      required this.SubRouteUID,
      required this.SubRouteName,
      required this.Direction,
      required this.EncodedPolyline,
      required this.UpdateTime,
    }
  );
  factory BusShape.fromJson(Map<String,dynamic> subroute){
    return BusShape(
      SubRouteUID: subroute["SubRouteUID"],
      SubRouteName: Map<String,String>.from(subroute["SubRouteName"]),
      Direction: subroute["Direction"],
      EncodedPolyline: subroute["EncodedPolyline"],
      UpdateTime: DateTime.parse(subroute["UpdateTime"]),
    );
  }
}