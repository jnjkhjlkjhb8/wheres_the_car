import 'package:flutter/material.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:animated_text_kit/animated_text_kit.dart';
import '../utility/Favorite.dart';

class HomePage extends StatefulWidget{
  const HomePage({super.key});
  @override
  State<HomePage> createState() => _HomePageState();
}

class _HomePageState extends State<HomePage> {
  final List<String> Greeting = [
    "Ciallo～(∠・ω< )⌒☆",
    "午安！希望你有個愉快的下午",
    "晚上好！祝你有個美好的夜晚"
  ];
  List<AnimatedText> get animatedTexts {
    final hour = DateTime.now().hour;
    if (hour < 12) {
      return [TypewriterAnimatedText(Greeting[0], textStyle: TextStyle(fontSize: 25, fontWeight: FontWeight.bold,color: Colors.black),speed: Duration(milliseconds: 100),)];
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
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    SizedBox(
                      height: 50,
                      child: AnimatedTextKit(
                        animatedTexts: animatedTexts,
                        repeatForever: true,
                      ),
                    )
                  ],
                ),
              )
            ),
            Padding(
              padding: const EdgeInsets.only(left:15.0,top: 10),
              child: Column(
                children: [
                  Text("我的最愛",style: TextStyle(color: Colors.grey[600],fontWeight: FontWeight.bold,fontSize: 18)),
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
          ]
        )
      );
  }
}