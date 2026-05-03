import 'package:hive/hive.dart';
import 'package:hive_flutter/hive_flutter.dart';

class Database{
  List past = [];
  late final Box<dynamic> _myBox;
  Database(){
    _myBox = Hive.box("myBox");
  }
  void saveData(String index, String temp){
    _myBox.put(index, temp);
  }
  String? getData(String index){
    return _myBox.get(index);
  }
  void deleteData(String index){
    _myBox.delete(index);
  }
  void updateData(String index,String temp){
    _myBox.delete(index);
    _myBox.put(index,temp);
  }
}