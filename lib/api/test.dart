import 'package:bus/data/BusEstimateTime.dart';
import 'package:flutter/services.dart';
Future<List<BusEstimates>> loadBusEstimates() async {
  final String response = await rootBundle.loadString('temp.json');
  return busEstimatesFromJson(response);
}
