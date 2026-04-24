import 'package:dio/dio.dart';
import 'dart:convert';
import '../utility/database.dart';
import '../data/DailyRoute.dart';
import 'package:pretty_dio_logger/pretty_dio_logger.dart';

List<Routes> allRoutes = [];
class UpdateRoute{
  final Dio _dio = Dio(
    BaseOptions(
      connectTimeout: Duration(seconds: 10),
      receiveTimeout: Duration(seconds: 10),
    ),
  )..interceptors.add(PrettyDioLogger());
  Future<void> fetchRoutes() async {
    Database db = Database();
    DateTime now = DateTime.now();
    final String today = "${now.year}-${now.month}-${now.day}";
    final String? cachedData = db.getData('cachedRoutes');
    final String? lastFetchTime = db.getData('lastFetchTime');
    if (cachedData == null || lastFetchTime != today) {
      try {
        final response = await _dio.get("https://raw.githubusercontent.com/jnjkhjlkjhb8/bus/refs/heads/main/routes.json?token=GHSAT0AAAAAADRRESMW2S7MJHLTWOTK56MQ2PLFENA");
        if (response.statusCode == 200) {
          String string = (response.data is String) ? response.data : jsonEncode(response.data);
          db.saveData('cachedRoutes', string);
          DateTime now = DateTime.now();
          db.saveData('lastFetchTime',today);
          allRoutes = routesFromJson(string);
          return;
        }
      }
      catch (e) {
        print ('Failed to load routes: $e');
        if (cachedData != null){
          allRoutes = routesFromJson(cachedData);
          return;
        }
      }
    }
    print("using local");
    allRoutes = routesFromJson(cachedData ?? "[]");
    return;
  }
}
class recent{
  Future<void> add(String RouteUID) async{
    final db = Database();
    String? recent = db.getData('recent');
    Map<String,dynamic> recentList = recent != null ? jsonDecode(recent) : {};
    recentList[RouteUID] = (recentList[RouteUID] ?? 0) + 1;
    db.saveData('recent', jsonEncode(recentList));
  }
  Future<void> del(String RouteUID) async{
    final db = Database();
    String? recent = db.getData('recent');
    Map<String,dynamic> recentList = recent != null ? jsonDecode(recent) : {};
    recentList.remove(RouteUID);
    db.saveData('recent', jsonEncode(recentList));
  }
  Future<Map<String, dynamic>> get() async{
    final db = Database();
    String? recent = db.getData('recent');
    Map<String,dynamic> recentList = recent != null ? jsonDecode(recent) : {};
    return recentList;
  }
}