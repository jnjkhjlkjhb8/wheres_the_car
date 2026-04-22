import 'dart:convert';

List<Routes> routesFromJson(String str) => List<Routes>.from(json.decode(str).map((x) => Routes.fromJson(x)));
String routesToJson(List<Routes> routes) => json.encode(List<dynamic>.from(routes.map((x) => x.toJson())));

class Routes{
  final String City;
  final String? DepartureStopNameZh;
  final String? DestinationStopNameZh;
  final String RouteID;
  final String? RouteMapImageUrl;
  final String RouteName;
  final String RouteUID;
  final int Type;
  Routes({
    required this.City,
    required this.DepartureStopNameZh,
    required this.DestinationStopNameZh,
    required this.RouteID,
    required this.RouteMapImageUrl,
    required this.RouteName,
    required this.RouteUID,
    required this.Type,
  });
  factory Routes.fromJson(Map<String, dynamic> json) {
    return Routes(
      City: json['City'],
      DepartureStopNameZh: json['DepartureStopNameZh'],
      DestinationStopNameZh: json['DestinationStopNameZh'],
      RouteID: json['RouteID'],
      RouteMapImageUrl: json['RouteMapImageUrl'],
      RouteName: json['RouteName'],
      RouteUID: json['RouteUID'],
      Type: json['Type'],
    );
  }
  Map<String, dynamic> toJson() => {
      'City': City,
      'DepartureStopNameZh': DepartureStopNameZh,
      'DestinationStopNameZh': DestinationStopNameZh,
      'RouteID': RouteID,
      'RouteMapImageUrl': RouteMapImageUrl,
      'RouteName': RouteName,
      'RouteUID': RouteUID,
      'Type': Type,
  };
}