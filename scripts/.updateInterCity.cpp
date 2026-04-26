#include<iostream>
#include<string>
#include<fstream>
#include<vector>
#include"nlohmann/json.hpp"
static nlohmann::json total = nlohmann::json::array();
static std::vector<std::string> city = {"KinmenCounty","PenghuCounty","LienchiangCounty"};
void getcity() {
    for (int i = 0;i < 3;i++) {
        std::string s = "curl 'https://tdx.transportdata.tw/api/basic/v2/Bus/Route/City/"+city[i]+"?$select=HasSubRoutes,RouteUID,RouteID,RouteName,BusRouteType,DepartureStopNameZh,DestinationStopNameZh,SubRoutes&$format=JSON' \
          -H 'accept: application/json' \
          -H 'accept-language: zh-TW,zh;q=0.9,en-US;q=0.8,en;q=0.7' \
          -H 'cache-control: max-age=0' \
          -H 'dnt: 1' \
          -H 'priority: u=0, i' \
          -H 'sec-ch-ua-mobile: ?0' \
          -H 'sec-fetch-dest: document' \
          -H 'sec-fetch-mode: navigate' \
          -H 'sec-fetch-site: none' \
          -H 'sec-fetch-user: ?1' \
          -H 'upgrade-insecure-requests: 1' \
          -H 'user-agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36' > temp1.json";
        system(s.c_str());
        std::ifstream f("temp1.json");
        if (f.good()) {
            nlohmann::json json = nlohmann::json::parse(f);
            if (json.is_array()) {
                for (auto &j : json) {
                    if (j["HasSubRoutes"] == true) {
                        for (auto &k : j["SubRoutes"]) {
                            nlohmann::json temp;
                            temp["RouteUID"] = j["RouteUID"];
                            temp["RouteName"] = j["RouteName"]["Zh_tw"];
                            temp["City"] = city[i];
                            temp["Type"] = j["BusRouteType"];
                            temp["SubRouteUID"] = k["SubRouteUID"];
                            temp["SubRouteName"] = k["SubRouteName"]["Zh_tw"];
                            temp["Direction"] = k["Direction"];
                            temp["DestinationStopNameZh"] = k.contains("DestinationStopNameZh") ? k["DestinationStopNameZh"] : j["DestinationStopNameZh"];
                            temp["DepartureStopNameZh"] = k.contains("DepartureStopNameZh") ? k["DepartureStopNameZh"] : j["DepartureStopNameZh"];
                            total.emplace_back(temp);
                        }
                    }
                }
            }
            else std::cout << "rate\n";
        }
        f.close();
    }
    remove("temp1.json");
}
void getinter() {
    std::string s = "curl 'https://tdx.transportdata.tw/api/basic/v2/Bus/Route/InterCity?$select=HasSubRoutes,RouteUID,RouteID,RouteName,BusRouteType,DepartureStopNameZh,DestinationStopNameZh,SubRoutes&$format=JSON' \
                  -H 'accept: text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7' \
                  -H 'accept-language: zh-TW,zh;q=0.9,en-US;q=0.8,en;q=0.7' \
                  -H 'cache-control: max-age=0' \
                  -H 'dnt: 1' \
                  -H 'priority: u=0, i' \
                  -H 'sec-ch-ua-mobile: ?0' \
                  -H 'sec-fetch-dest: document' \
                  -H 'sec-fetch-mode: navigate' \
                  -H 'sec-fetch-site: none' \
                  -H 'sec-fetch-user: ?1' \
                  -H 'upgrade-insecure-requests: 1' \
                  -H 'user-agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36' > temp2.json";
    system(s.c_str());
    std::ifstream f("temp2.json");
    if (f.good()) {
        nlohmann::json json = nlohmann::json::parse(f);
        for (auto &j : json) {
            if (j["HasSubRoutes"] == true) {
                for (auto &k : j["SubRoutes"]) {
                    nlohmann::json temp;
                    temp["RouteUID"] = j["RouteUID"];
                    temp["RouteName"] = j["RouteName"]["Zh_tw"];
                    temp["City"] = "InterCity";
                    temp["Type"] = j["BusRouteType"];
                    temp["SubRouteUID"] = k["SubRouteUID"];
                    temp["SubRouteName"] = k["SubRouteName"]["Zh_tw"];
                    temp["Direction"] = k["Direction"];
                    temp["DestinationStopNameZh"] = k.contains("DestinationStopNameZh") ? k["DestinationStopNameZh"] : j["DestinationStopNameZh"];
                    temp["DepartureStopNameZh"] = k.contains("DepartureStopNameZh") ? k["DepartureStopNameZh"] : j["DepartureStopNameZh"];
                    total.emplace_back(temp);
                }
            }
        }
    }
    f.close();
    remove("temp2.json");
}
int main() {
    try {
        getcity();
        getinter();
        std::ofstream out("routes2.json");
        out << total.dump();
        out.close();
    }
    catch (std::exception &e) {
        std::cerr << "error: " << e.what() << '\n';
    }
}