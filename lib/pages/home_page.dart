import 'package:flutter/material.dart';

class HomePage extends StatefulWidget{
  const HomePage({super.key});
  @override
  State<HomePage> createState() => _HomePageState();
}

class _HomePageState extends State<HomePage> {
  int currentPage = 0;
  @override
  Widget build(BuildContext context){
      return Scaffold(
        appBar: AppBar(
          title: const Text("首頁"),
        ),
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
            NavigationDestination(icon: Icon(Icons.public), label: "地圖"),
            NavigationDestination(icon: Icon(Icons.train), label: "台鐵"),
            NavigationDestination(icon: Icon(Icons.settings), label: "設定"),
          ],
        ),
      );
  }
}