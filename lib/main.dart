import 'package:bus/pages/home_page.dart';
import '../pages/map_page.dart';
import 'package:flutter/material.dart';
import 'package:hive_flutter/hive_flutter.dart';

void main() async {
  await Hive.initFlutter();
  await Hive.openBox("myBox");
  runApp(const MyApp());
}

class MyApp extends StatelessWidget {
  const MyApp({super.key});
  @override
  Widget build(BuildContext context){
    return MaterialApp(
      theme: ThemeData(
        useMaterial3: true,
        colorScheme: ColorScheme.fromSeed(seedColor: Colors.blue),
      ),
      home: const MainScreen(),
    );
  }
}

class MainScreen extends StatefulWidget{
  const MainScreen({super.key});
  @override
  State<MainScreen> createState() => _MainScreenState();
}

class _MainScreenState extends State<MainScreen> {
  int currentPage = 0;
  final List<Widget> pages = [
    const HomePage(),
    const Center(child: Text("捷運頁面")),
    const MapPage(),
    const Center(child: Text("雙鐵頁面")),
    const Center(child: Text("設定頁面")),
  ];
  @override
  Widget build(BuildContext context){
    return Scaffold(
      body: pages[currentPage],
      bottomNavigationBar: NavigationBar(
          height: 67,
          onDestinationSelected: (int index){
            setState((){
              currentPage = index;
            });
          },
          selectedIndex: currentPage,
          destinations: [
            NavigationDestination(icon: Icon(Icons.home), label: "首頁"),
            NavigationDestination(icon: Icon(Icons.subway), label: "捷運"),
            NavigationDestination(icon: Icon(Icons.map), label: "地圖"),
            NavigationDestination(icon: Icon(Icons.train), label: "雙鐵"),
            NavigationDestination(icon: Icon(Icons.settings), label: "設定"),
          ],
        ),
    );
  }
}