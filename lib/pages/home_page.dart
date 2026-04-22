import 'package:bus/data/DailyRoute.dart';
import 'package:flutter/material.dart';
import 'package:animated_text_kit/animated_text_kit.dart';
import 'package:shared_preferences/shared_preferences.dart';
import '../utility/Favorite.dart';
import '../utility/update_route.dart';

final Map<String, String> _cites = {
    "Taipei": "台北市",
    "NewTaipei": "新北市",
    "Taoyuan": "桃園市",
    "Taichung": "台中市",
    "Tainan": "台南市",
    "Kaohsiung": "高雄市",
    "Keelung": "基隆市",
    "Hsinchu": "新竹市",
    "HsinchuCounty": "新竹縣",
    "MiaoliCounty": "苗栗縣",
    "ChanghuaCounty": "彰化縣",
    "NantouCounty": "南投縣",
    "YunlinCounty": "雲林縣",
    "ChiayiCounty": "嘉義縣",
    "Chiayi": "嘉義市",
    "PingtungCounty": "屏東縣", 
    "YilanCounty": "宜蘭縣",
    "HualienCounty": "花蓮縣",
    "TaitungCounty": "台東縣",
    "KinmenCounty": "金門縣",
    "PenghuCounty": "澎湖縣",
    "LienchiangCounty": "連江縣"
};
class HomePage extends StatefulWidget{
  const HomePage({super.key});
  @override
  State<HomePage> createState() => _HomePageState();
}

class _HomePageState extends State<HomePage> {
  final UpdateRoute fetchRoutes = UpdateRoute();
  final SearchController _searchController = SearchController();
  final List<String> Greeting = [
    "Ciallo～(∠・ω< )⌒☆",
    "test",
    "test2",
  ];
  List<AnimatedText> get animatedTexts {
    final hour = DateTime.now().hour;
    if (hour < 12) {
      return [TypewriterAnimatedText(Greeting[0], textStyle: TextStyle(fontSize: 25, fontWeight: FontWeight.bold,color: Colors.black),speed: Duration(milliseconds: 100))];
    } else if (hour < 18) {
      return [TypewriterAnimatedText(Greeting[1], textStyle: TextStyle(fontSize: 25, fontWeight: FontWeight.bold,color: Colors.black),speed: Duration(milliseconds: 100),)];
    } else {
      return [TypewriterAnimatedText(Greeting[2], textStyle: TextStyle(fontSize: 25, fontWeight: FontWeight.bold,color: Colors.black),speed: Duration(milliseconds: 100), )];
    }
  }
  int currentPage = 0;
  @override
  Widget build(BuildContext context){
    final colorscheme = Theme.of(context).colorScheme;
      return Scaffold(
        body: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Container(
              height: 100,
              width: double.infinity,
              decoration: BoxDecoration(
                color: colorscheme.primaryContainer,
                borderRadius: BorderRadius.only(bottomRight: Radius.circular(33.5), bottomLeft: Radius.circular(33.5)),
              ),
              child: Padding(
                padding: const EdgeInsets.fromLTRB(30, 50, 24, 0),
                child: Center(
                  child: SizedBox(
                      height: 50,
                      child: AnimatedTextKit(
                        animatedTexts: animatedTexts,
                        repeatForever: true,
                      ),
                    )
                ),
              )
            ),
            Padding(
              padding: const EdgeInsets.only(left:15.0,top: 10),
              child: Column(
                children: [
                  Text("我的最愛",style: TextStyle(color: Colors.grey[600],fontWeight: FontWeight.bold,fontSize: 18,)),
                ],
              ),
            ),
            ConstrainedBox(
              constraints: BoxConstraints(maxHeight: 200),
              child: FutureBuilder<List<Favorite>>(
                future: getFavorites(),
                builder: (context, snapshot) {
                  if (snapshot.connectionState == ConnectionState.waiting) return const Center(child: CircularProgressIndicator());
                  else if (snapshot.hasError) return const Center(child: Text('發生錯誤，無法載入最愛路線',style: TextStyle(fontWeight: FontWeight.bold)));
                  else if (snapshot.data!.isEmpty) return const Center(child: Text('尚未新增過任何東西',style: TextStyle(fontWeight: FontWeight.bold)));
                  else {
                    final favorites = snapshot.data!;
                    return ListView.builder(
                      shrinkWrap: true,
                      itemCount: favorites.length,
                      itemBuilder: (context, index) {
                        final favorite = favorites[index];
                        return Dismissible(
                          key: Key(favorite.route),
                          direction: DismissDirection.endToStart,
                          background: Container(
                            color: Colors.red,
                            alignment: Alignment.centerRight,
                            padding: EdgeInsets.symmetric(horizontal: 20),
                            child: Icon(Icons.delete, color: Colors.white),
                          ),
                          onDismissed: (direction) async {
                            await delFavorites(index, favorites);
                            setState(() {});
                          },
                          child: ListTile(
                            title: Text(favorite.route),
                            subtitle: Text('${favorite.StartStaion} → ${favorite.EndStation}'),
                          ),
                        );
                      },
                    );
                  }
                }
              )
            ),
            const SizedBox(height: 20,child: Divider(thickness: 1.67,)),
            Padding(
              padding: EdgeInsets.all(15),
              child: SearchAnchor(
                searchController: _searchController,
                builder: (context, controller){
                  return SearchBar(
                    controller: controller,
                    hintText: "搜尋路線",
                    leading: Icon(Icons.search),
                    onTap: () => controller.openView(),
                  );
                },
                suggestionsBuilder: (context,controller) async {
                  if (allRoutes.isEmpty) await fetchRoutes.fetchRoutes();
                  final input = controller.text.trim();
                  Map<String,dynamic> recentList = await recent().get();
                  if (input.isEmpty){
                    if (recentList.isEmpty) return [const ListTile(title: Text("沒有最近搜尋紀錄"))];
                    List<Routes> recentRoutes = allRoutes.where((route) => recentList.containsKey(route.RouteUID)).toList();
                    recentRoutes.sort((a, b) => recentList[b.RouteUID].compareTo(recentList[a.RouteUID]));
                    return recentRoutes.map((route) => ListTile(
                      title: Text.rich(
                        TextSpan(
                          text: route.RouteName,
                          style: TextStyle(fontWeight: FontWeight.bold),
                          children: [
                            WidgetSpan(child: const SizedBox(width: 5)),
                            TextSpan(
                              text: _cites[route.City],
                              style: TextStyle(fontSize: 12, color: Colors.grey[600], fontWeight: FontWeight.normal),
                            ),
                          ],
                        ),
                      ),
                      subtitle: Text("${route.DepartureStopNameZh} ⇌ ${route.DestinationStopNameZh}"),
                      trailing: (
                        IconButton(
                          icon: Icon(Icons.delete, color: Colors.red),
                          onPressed: () async {
                            showDialog(
                              context: context,
                              builder: (context) => AlertDialog(
                                title: Text("確定要刪除這項紀錄嗎？",style: TextStyle(fontWeight: FontWeight.bold)),
                                content: Text.rich(
                                  TextSpan(
                                    text: route.RouteName,
                                    style: TextStyle(fontWeight: FontWeight.bold),
                                    children: [
                                      WidgetSpan(child: const SizedBox(width: 5)),
                                      TextSpan(
                                        text: _cites[route.City],
                                        style: TextStyle(fontSize: 12, color: Colors.grey[600], fontWeight: FontWeight.normal),
                                      ),
                                    ],
                                  ),
                                ),
                                actions: [
                                  FilledButton.tonal(
                                    onPressed: () => Navigator.pop(context),
                                    child: Text("取消"),
                                  ),
                                  FilledButton(
                                    onPressed: () async {
                                      await recent().del(route.RouteUID);
                                      setState(() {});
                                      Navigator.pop(context);
                                    },
                                    child: Text("確定"),
                                  ),
                                ],
                              ),
                            );
                            setState(() {});
                          }
                        )
                      ),
                      onTap: () {
                        controller.text = route.RouteName;
                        controller.closeView(null);
                        recent().add(route.RouteUID);
                        setState(() {}); // 
                      },
                    )).toList();
                    
                  }
                  final foundRoutes = allRoutes.where((route) => route.RouteName.contains(input)).toList();
                  if (foundRoutes.isEmpty) return [const ListTile(title: Text("沒有找到相關路線"))];
                  Map<String, List<Routes>> city = {};
                  List<Widget> Display = [];
                  for (var route in foundRoutes) city.putIfAbsent(route.City, () => []).add(route);
                  city.forEach((cityKey, routes) {
                    Display.add(
                      Padding(
                        padding: const EdgeInsets.all(25),
                        child: Text(_cites[cityKey] ?? '',style: TextStyle(fontSize: 18, fontWeight: FontWeight.bold, color: colorscheme.primary),),
                      )
                    );
                    Display.addAll(
                      routes.map((route) => ListTile(
                        title: Text.rich(
                          TextSpan(
                            text: route.RouteName,
                            style: TextStyle(fontWeight: FontWeight.bold),
                            children: [
                              WidgetSpan(child: const SizedBox(width: 5)),
                              TextSpan(
                                text: _cites[route.City],
                                style: TextStyle(fontSize: 12, color: Colors.grey[600], fontWeight: FontWeight.normal),
                              ),
                            ],
                          ),
                        ),
                        subtitle: Text("${route.DepartureStopNameZh} ⇌ ${route.DestinationStopNameZh}"),
                        onTap: () {
                          controller.text = route.RouteName;
                          controller.closeView(null);
                          recent().add(route.RouteUID);
                          setState(() {});
                        },
                      )).toList()
                    );
                  });
                  return Display;
                },
              )
            )
          ]
        )
      );
  }
}