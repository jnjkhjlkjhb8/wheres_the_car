import 'dart:convert';
import '../utility/database.dart';

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
  final Database db = Database();
  final String? json = db.getData("favorites");
  final List<Map<String, dynamic>> favoritesJson = favorites.map((favorite) => favorite.toJson()).toList();
  db.saveData('favorites', jsonEncode(favoritesJson));
}
Future<List<Favorite>> getFavorites() async{
    final Database db = Database();
    final String? json = db.getData("favorites");
    if (json == null) return [];
    List<Favorite> list = favoritesFromJson(json);
    return list;
}
Future<void> delFavorites(int index,List<Favorite> favorites) async{
    final Database db = Database();
    favorites.removeAt(index);
    final List<Map<String, dynamic>> favoritesJson = favorites.map((favorite) => favorite.toJson()).toList();
    db.saveData('favorites', jsonEncode(favoritesJson));
}