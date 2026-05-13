import 'dart:convert';

List<Routes> routesFromJson(String str) => List<Routes>.from(json.decode(str).map((x) => Routes.fromJson(x)));
String routesToJson(List<Routes> routes) => json.encode(List<dynamic>.from(routes.map((x) => x.toJson())));

class Routes{
  final String City;
  final String? DepartureStopNameZh;
  final String? DestinationStopNameZh;
  final String? RouteID;
  final String RouteName;
  final String RouteUID;
  final String? SubRouteUID;
  final String SubRouteName;
  final int Type;
  Routes({
    required this.City,
    required this.DepartureStopNameZh,
    required this.DestinationStopNameZh,
    required this.RouteID,
    required this.RouteName,
    required this.RouteUID,
    required this.Type,
    required this.SubRouteUID,
    required this.SubRouteName,
  });
  factory Routes.fromJson(Map<String, dynamic> json) {
    return Routes(
      City: json['City'],
      DepartureStopNameZh: json['DepartureStopNameZh'],
      DestinationStopNameZh: json['DestinationStopNameZh'],
      RouteID: json['RouteID'],
      RouteName: json['RouteName'],
      RouteUID: json['RouteUID'],
      Type: json['Type'],
      SubRouteUID: json['SubRouteUID'],
      SubRouteName: json['SubRouteName'],
    );
  }
  Map<String, dynamic> toJson() => {
      'City': City,
      'DepartureStopNameZh': DepartureStopNameZh,
      'DestinationStopNameZh': DestinationStopNameZh,
      'RouteID': RouteID,
      'RouteName': RouteName,
      'RouteUID': RouteUID,
      'Type': Type,
      'SubRouteUID': SubRouteUID,
      'SubRouteName': SubRouteName,
  };
}