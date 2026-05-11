import 'dart:convert';

List<Busdailystops> BusdailystopsFromJson(dynamic str) {
  final List<dynamic> jsonData = str is String ? json.decode(str) : str;
  return List<Busdailystops>.from(jsonData.map((x) => Busdailystops.fromJson(x)));
}

class TimeTable{
  final int Sequence;
  final bool IsLowFloor;
  final DateTime DepartureTime;
  TimeTable({
    required this.Sequence,
    required this.IsLowFloor,
    required this.DepartureTime,
  });
  factory TimeTable.fromJson(Map<String, dynamic> json) {
    String temp = json['DepartureTime'] ?? "00:00";
    List<String> temp2 = temp.split(':');
    DateTime now = DateTime.now();
    DateTime res = DateTime(now.year, now.month, now.day, int.parse(temp2[0]), int.parse(temp2[1]));
    return TimeTable(
      Sequence: json['Sequence'],
      IsLowFloor: json['IsLowFloor'] ?? false,
      DepartureTime: res,
    );
  }
}
class Stop{
  final String StopUID;
  final Map<String,String> StopName;
  final List<TimeTable> TimeTables;
  Stop({
    required this.StopUID,
    required this.StopName,
    required this.TimeTables,
  });
  factory Stop.fromJson(Map<String, dynamic> json) {
    return Stop(
      StopUID: json['StopUID'],
      StopName: Map<String,String>.from(json['StopName']),
      TimeTables: List<TimeTable>.from(json['TimeTables'].map((x) => TimeTable.fromJson(x))),
    );
  }
}
class Busdailystops {
  final DateTime BusDate;
  final String RouteUID;
  final String SubRouteUID;
  final Map<String,String> SubRouteName;
  final List<Stop> Stops;
  Busdailystops({
    required this.BusDate,
    required this.RouteUID,
    required this.SubRouteUID,
    required this.SubRouteName,
    required this.Stops,
  });
  factory Busdailystops.fromJson(Map<String, dynamic> json) {
    return Busdailystops(
      BusDate: DateTime.parse(json['BusDate']),
      RouteUID: json['RouteUID'],
      SubRouteUID: json['SubRouteUID'],
      SubRouteName: Map<String, String>.from(json['SubRouteName']),
      Stops: List<Stop>.from(json['Stops'].map((x) => Stop.fromJson(x))),
    );
  }
}