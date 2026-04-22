import 'package:dio/dio.dart';
import 'dart:convert';
import 'package:shared_preferences/shared_preferences.dart';
import '../data/DailyRoute.dart';

List<Routes> allRoutes = [];
class UpdateRoute{
  final Dio _dio = Dio(
    BaseOptions(
      baseUrl: "https://raw.githubusercontent.com/jnjkhjlkjhb8/bus/refs/heads/main/routes.json?token=GHSAT0AAAAAADRRESMWCZZKSVWW7ONYJFH62PJAO7Q",
      connectTimeout: Duration(seconds: 10),
      receiveTimeout: Duration(seconds: 10),
    ),
  );
  Future<void> fetchRoutes() async {
    final prefs = await SharedPreferences.getInstance();
    DateTime now = DateTime.now();
    final String today = "${now.year}-${now.month}-${now.day}";
    final String? cachedData = prefs.getString('cachedRoutes');
    final String? lastFetchTime = prefs.getString('lastFetchTime');
    if (cachedData == null || lastFetchTime != today) {
      try {
        final response = await _dio.get("https://raw.githubusercontent.com/jnjkhjlkjhb8/bus/refs/heads/main/routes.json?token=GHSAT0AAAAAADRRESMWCZZKSVWW7ONYJFH62PJAO7Q");
        if (response.statusCode == 200) {
          String string = (response.data is String) ? response.data : jsonEncode(response.data);
          await prefs.setString('cachedRoutes', string);
          DateTime now = DateTime.now();
          await prefs.setString('lastFetchTime',today);
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
    final prefs = await SharedPreferences.getInstance();
    String? recent = prefs.getString('recent');
    Map<String,dynamic> recentList = recent != null ? jsonDecode(recent) : {};
    recentList[RouteUID] = (recentList[RouteUID] ?? 0) + 1;
    await prefs.setString('recent', jsonEncode(recentList));
  }
  Future<void> del(String RouteUID) async{
    final prefs = await SharedPreferences.getInstance();
    String? recent = prefs.getString('recent');
    Map<String,dynamic> recentList = recent != null ? jsonDecode(recent) : {};
    recentList.remove(RouteUID);
    await prefs.setString('recent', jsonEncode(recentList));
  }
  Future<Map<String, dynamic>> get() async{
    final prefs = await SharedPreferences.getInstance();
    String? recent = prefs.getString('recent');
    Map<String,dynamic> recentList = recent != null ? jsonDecode(recent) : {};
    return recentList;
  }
}