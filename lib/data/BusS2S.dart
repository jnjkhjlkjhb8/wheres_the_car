import 'dart:convert';

List<BusS2S> busS2SFromJson(dynamic str) {
  final List<dynamic> jsonData = str is String ? json.decode(str) : str;
  return List<BusS2S>.from(jsonData.map((x) => BusS2S.fromJson(x)));
}

class S2STime{
  final String FromStopID;
  final String ToStopID;
  final String FromStationID;
  final String ToStationID;
  final int RunTime;
  S2STime({
    required this.FromStopID,
    required this.ToStopID,
    required this.FromStationID,
    required this.ToStationID,
    required this.RunTime
  });
  factory S2STime.fromJson(Map<String, dynamic> json) {
    return S2STime(
      FromStopID: json['FromStopID'],
      ToStopID: json['ToStopID'],
      FromStationID: json['FromStationID'],
      ToStationID: json['ToStationID'],
      RunTime: json['RunTime'],
    );
  }
}
class TravelTime{
  final int Weekday;
  final int StartHour;
  final int EndHour;
  final List<S2STime> S2STimes;
  TravelTime({
    required this.Weekday,
    required this.StartHour,
    required this.EndHour,
    required this.S2STimes
  });
  factory TravelTime.fromJson(Map<String, dynamic> json) {
    return TravelTime(
      Weekday: json['Weekday'],
      StartHour: json['StartHour'],
      EndHour: json['EndHour'],
      S2STimes: (json["S2STimes"] as List<dynamic>?)
          !.map((e) => S2STime.fromJson(Map<String, dynamic>.from(e)))
          .toList(),
    );
  }
}
class BusS2S{
  final String RouteUID;
  final String SubRouteUID;
  final String SubRouteID;
  final int Direction;
  final List<TravelTime> TravelTimes;
  BusS2S({
    required this.RouteUID,
    required this.SubRouteUID,
    required this.SubRouteID,
    required this.Direction,
    required this.TravelTimes
  });
  factory BusS2S.fromJson(Map<String, dynamic> json) {
    return BusS2S(
      RouteUID: json['RouteUID'],
      SubRouteUID: json['SubRouteUID'],
      SubRouteID: json['SubRouteID'],
      Direction: json['Direction'],
      TravelTimes: (json["TravelTimes"] as List<dynamic>?)
      !.map((e) => TravelTime.fromJson(Map<String, dynamic>.from(e)))
          .toList(),
    );
  }
}