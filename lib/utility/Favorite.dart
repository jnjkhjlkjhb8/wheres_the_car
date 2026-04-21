import 'dart:convert';
import 'package:shared_preferences/shared_preferences.dart';

List<Favorite> favoritesFromJson(String str) => List<Favorite>.from(json.decode(str).map((x) => Favorite.fromJson(x))).toList();

class Favorite{
  final String city;
  final String route;
  final String StartStaion;
  final String EndStation;
  Favorite({required this.city, required this.route, required this.StartStaion, required this.EndStation});
  factory Favorite.fromJson(Map<String, dynamic> json) {
    return Favorite(
      city: json['city'],
      route: json['route'],
      StartStaion: json['StartStaion'],
      EndStation: json['EndStation'],
    );
  }
  Map<String, dynamic> toJson() {
    return {
      'city': city,
      'route': route,
      'StartStaion': StartStaion,
      'EndStation': EndStation,
    };
  }
}
Future <void> saveFavorite(List<Favorite> favorites) async {
  final prefs = await SharedPreferences.getInstance();
  final List<Map<String, dynamic>> favoritesJson = favorites.map((favorite) => favorite.toJson()).toList();
  await prefs.setString('favorites', jsonEncode(favoritesJson));
}
Future<List<Favorite>> getFavorites() async{
    final prefs = await SharedPreferences.getInstance();
    final String? json = prefs.getString('favorites');
    if (json == null) return [];
    List<Favorite> list = favoritesFromJson(json);
    return list;
}
Future<void> delFavorites(int index,List<Favorite> favorites) async{
    final prefs = await SharedPreferences.getInstance();
    favorites.removeAt(index);
    String temp = jsonEncode(favorites.map((favorite) => favorite.toJson()).toList());
    prefs.setString("favorites", temp);
}