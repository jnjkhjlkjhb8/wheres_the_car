#include<iostream>
#include<string>
#include<fstream>
#include<vector>
#include<thread>
#include<chrono>
#include"json.hpp"
static nlohmann::json total = nlohmann::json().array();
std::string token = "eyJhbGciOiJSUzI1NiIsInR5cCIgOiAiSldUIiwia2lkIiA6ICJER2lKNFE5bFg4WldFajlNNEE2amFVNm9JOGJVQ3RYWGV6OFdZVzh3ZkhrIn0.eyJleHAiOjE3NzY3ODc2OTUsImlhdCI6MTc3NjcwMTI5NSwianRpIjoiYjRkNTcxZjMtMGNhZC00ODA2LWE2OWMtYWZiMGJlNmVmMzdlIiwiaXNzIjoiaHR0cHM6Ly90ZHgudHJhbnNwb3J0ZGF0YS50dy9hdXRoL3JlYWxtcy9URFhDb25uZWN0Iiwic3ViIjoiNWZlNWYxYjEtMTFiZS00NjliLWFlMTItZjkzYTI1ZmU3ZGM3IiwidHlwIjoiQmVhcmVyIiwiYXpwIjoiNDExMjExLTIyZGNjMTQ2LTE3ZWQtNDI0ZiIsImFjciI6IjEiLCJyZWFsbV9hY2Nlc3MiOnsicm9sZXMiOlsic3RhdGlzdGljIiwicHJlbWl1bSIsInBhcmtpbmdGZWUiLCJtYWFzIiwiYWR2YW5jZWQiLCJnZW9pbmZvIiwidmFsaWRhdG9yIiwidG91cmlzbSIsImhpc3RvcmljYWwiLCJjd2EiLCJiYXNpYyJdfSwic2NvcGUiOiJwcm9maWxlIGVtYWlsIiwidXNlciI6ImY4NmIwYWI5In0.qcGcUKjnT_ml8EFFohtB9oVMm1IQWS8iZqKZOCFIFI3__ouyVIO8xfgRgVSJJJLriXcXrw79GabKfE-agn-rp0bjyt8m0iCxMFZ_7EKDWnfVwOPFehrgdLPDTripB5DmsmF9j_VUbZth5sewxTRPr_mKqFFwWYKQ7Vb0nNwY0A9M6jpVoXXiikbK22ItpInKYeiU28tqxSPv89HFQuCohy8nm2whmRpdIIsI64quWA3rjdBwNxTdf44gYZONra2XXIYdDSVqsenBHhv3qV2B9b9IV81sgUV2p97NYO8I2BCgI7WZhY2wSv_fvtfl6681H_yqX1tJkaIETqDV7yKdPA";
std::vector<std::string> city = {"Taipei","NewTaipei","Taoyuan","Taichung","Tainan","Kaohsiung","Keelung","Hsinchu", "HsinchuCounty","MiaoliCounty","ChanghuaCounty","NantouCounty","YunlinCounty","ChiayiCounty","Chiayi","PingtungCounty","YilanCounty","HualienCounty","TaitungCounty","KinmenCounty","PenghuCounty","LienchiangCounty"};
void gettoken(const char *id,const char *secret) { // Bug 可能是secrets
    std::string s = "curl --request POST https://tdx.transportdata.tw/auth/realms/TDXConnect/protocol/openid-connect/token "
                    "--header 'content-type: application/x-www-form-urlencoded' "
                    "--data 'grant_type=client_credentials&client_id=" + std::string(id) +
                    "&client_secret="+ std::string(secret) + "' > token.json";
    system(s.c_str());
    std::ifstream f("token.json");
    nlohmann::json j = nlohmann::json::parse(f);
    if (j.contains("access_token")) token = j["access_token"];
    f.close();
    remove("token.json");
    std::cerr << token << '\n';
}
void getcity() {
    for (int i = 0;i < 22;i++) {
        std::string s = "curl -X 'GET' "
                        "-H 'authorization: Bearer " + token + "' " +
                        "-H 'Content-Encoding: br,gzip' " +
                        "-H 'Content-Type: application/json' " +
                        "\'https://tdx.transportdata.tw/api/basic/v2/Bus/Route/City/" +city[i] +"?$select=RouteUID,RouteID,RouteName,BusRouteType,DepartureStopNameZh,DestinationStopNameZh&$format=JSON\' > temp.json";
        system(s.c_str());
        std::ifstream f("temp.json");
        if (f.good()) {
            nlohmann::json json = nlohmann::json::parse(f);
            if (json.is_array()) {
                for (auto &j : json) {
                    nlohmann::json temp;
                    temp["RouteUID"] = j["RouteUID"];
                    temp["RouteID"] = j["RouteID"];
                    temp["RouteName"] = j["RouteName"]["Zh_tw"];
                    temp["City"] = city[i];
                    temp["Type"] = j["BusRouteType"];
                    temp["DepartureStopNameZh"] = j["DepartureStopNameZh"];
                    temp["DestinationStopNameZh"] = j["DestinationStopNameZh"];
                    total.emplace_back(temp);
                }
            }
            else std::cout << "rate\n";
        }
        f.close();
        if ((i+1) % 5 == 0) {
            std::cout << "sleep\n";
            std::this_thread::sleep_for(std::chrono::seconds(60));
        }
    }
    remove("temp.json");
}
void getinter() {
    std::string s = "curl -X 'GET' "
                    "-H 'authorization: Bearer " + token + "' " +
                    "-H 'Content-Encoding: br,gzip' " +
                    "-H 'Content-Type: application/json' " +
                    "\'https://tdx.transportdata.tw/api/basic/v2/Bus/Route/InterCity?$select=RouteUID,RouteID,RouteName,BusRouteType,DepartureStopNameZh,DestinationStopNameZh&$format=JSON\' > temp.json";
    system(s.c_str());
    std::ifstream f("temp.json");
    if (f.good()) {
        if (f.good()) {
            nlohmann::json json = nlohmann::json::parse(f);
            for (auto &j : json) {
                nlohmann::json temp;
                temp["RouteUID"] = j["RouteUID"];
                temp["RouteName"] = j["RouteName"]["Zh_tw"];
                temp["RouteID"] = j["RouteID"];
                temp["City"] = "InterCity";
                temp["Type"] = j["BusRouteType"];
                temp["DepartureStopNameZh"] = j["DepartureStopNameZh"];
                temp["DestinationStopNameZh"] = j["DestinationStopNameZh"];
                total.emplace_back(temp);
            }
        }
    }
    f.close();
    remove("temp.json");
}
int main() {
    freopen("temp.json","w",stdout);
    const char *id = getenv("Client_ID"),*secret = getenv("Client_SECRET");
    std::cout << std::string(id) << ' ' << std::string(secret) << '\n';
    try {
        //gettoken(id, secret);
        getcity();
        getinter();
        std::ofstream out("routes.json");
        out << total.dump();
        out.close();
        std::cout << "test " << total.size() << " \n";
    }
    catch (std::exception &e) {
        std::cerr << "error: " << e.what() << '\n';
    }
    return 0;
}