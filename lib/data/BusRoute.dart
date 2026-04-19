import 'dart:convert';

List<BusRoute> BusRouteFromJson(String str) => List<BusRoute>.from(json.decode(str).map((x) => BusRoute.fromJson(x)));

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
class SubRoute{
  final String SubRouteUID;
  final String SubRouteID;
  final List<String> OperatorIDs;
  final Map<String,String> SubRouteName;
  final String Headsign;
  final String HeadsignEn;
  final int Direction;
  final String FirstBusTime;
  final String LastBusTime;
  final String HolidayFirstBusTime;
  final String HolidayLastBusTime;
  final String DepartureStopNameZh;
  final String DepartureStopNameEn;
  final String DestinationStopNameZh;
  final String DestinationStopNameEn;
  SubRoute(
    {
      required this.SubRouteUID,
      required this.SubRouteID,
      required this.OperatorIDs,
      required this.SubRouteName,
      required this.Headsign,
      required this.HeadsignEn,
      required this.Direction,
      required this.FirstBusTime,
      required this.LastBusTime,
      required this.HolidayFirstBusTime,
      required this.HolidayLastBusTime,
      required this.DepartureStopNameZh,
      required this.DepartureStopNameEn,
      required this.DestinationStopNameZh,
      required this.DestinationStopNameEn
    }
  );
  factory SubRoute.fromJson(Map<String,dynamic> subroute){
    return SubRoute(
      SubRouteUID: subroute["SubRouteUID"],
      SubRouteID: subroute["SubRouteID"],
      OperatorIDs: List<String>.from(subroute["OperatorIDs"]),
      SubRouteName: Map<String,String>.from(subroute["SubRouteName"]),
      Headsign: subroute["HeadSign"],
      HeadsignEn: subroute["HeadSignEn"],
      Direction: subroute["Direction"],
      FirstBusTime: subroute["FirstBusTime"],
      LastBusTime: subroute["LastBusTime"],
      HolidayFirstBusTime: subroute["HolidayFirstBusTime"],
      HolidayLastBusTime: subroute["HolidayLastBusTime"],
      DepartureStopNameZh: subroute["DepartureStopNameZh"],
      DepartureStopNameEn: subroute["DepartureStopNameEn"],
      DestinationStopNameZh: subroute["DestinationStopNameZh"],
      DestinationStopNameEn: subroute["DestinationStopNameEn"]
    );
  }
}
class BusRoute{
  final String RouteUID;
  final String RouteID;
  final bool HasSubRoutes;
  final List<BusOperators> Operators;
  final String AuthorityID;
  final String ProviderID;
  final int BusRouteType;
  final Map<String,String> RouteName;
  final String DepartureStopNameZh;
  final String DepartureStopNameEn;
  final String DestinationStopNameZh;
  final String DestinationStopNameEn;
  final String TicketPriceDescriptionZh;
  final String TicketPriceDescriptionEn;
  final String FareBufferZoneDescriptionZh;
  final String FareBufferZoneDescriptionEn;
  final String RouteMapImageUrl;
  final String City;
  final String CityCode;
  final DateTime UpdateTime;
  final int VersionID;
  BusRoute(
    {
      required this.RouteUID,
      required this.RouteID,
      required this.HasSubRoutes,
      required this.Operators,
      required this.AuthorityID,
      required this.ProviderID,
      required this.BusRouteType,
      required this.RouteName,
      required this.DepartureStopNameZh,
      required this.DepartureStopNameEn,
      required this.DestinationStopNameZh,
      required this.DestinationStopNameEn,
      required this.TicketPriceDescriptionZh,
      required this.TicketPriceDescriptionEn,
      required this.FareBufferZoneDescriptionZh,
      required this.FareBufferZoneDescriptionEn,
      required this.RouteMapImageUrl,
      required this.City,
      required this.CityCode,
      required this.UpdateTime,
      required this.VersionID
    }
  );
  factory BusRoute.fromJson(Map<String,dynamic> route){
    return BusRoute(
      RouteUID: route["RouteUID"],
      RouteID: route["RouteID"],
      HasSubRoutes: route["HasSubRoutes"],
      Operators: (route["Operators"] as List)
        .map((operator) => BusOperators.fromJson(operator))
        .toList(),
      AuthorityID: route["AuthorityID"],
      ProviderID: route["ProviderID"],
      BusRouteType: route["BusRouteType"],
      RouteName: Map<String,String>.from(route["RouteName"]),
      DepartureStopNameZh: route["DepartureStopNameZh"],
      DepartureStopNameEn: route["DepartureStopNameEn"],
      DestinationStopNameZh: route["DestinationStopNameZh"],
      DestinationStopNameEn: route["DestinationStopNameEn"],
      TicketPriceDescriptionZh: route["TicketPriceDescriptionZh"],
      TicketPriceDescriptionEn: route["TicketPriceDescriptionEn"],
      FareBufferZoneDescriptionZh: route["FareBufferZoneDescriptionZh"],
      FareBufferZoneDescriptionEn: route["FareBufferZoneDescriptionEn"],
      RouteMapImageUrl: route["RouteMapImageUrl"],
      City: route["City"],
      CityCode: route["CityCode"],
      UpdateTime: DateTime.parse(route["UpdateTime"]),
      VersionID: route["VersionID"]
    );
  }
}