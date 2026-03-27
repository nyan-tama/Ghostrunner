import SwiftUI

struct ContentView: View {
    var body: some View {
        VStack(spacing: 16) {
            Image(systemName: "macwindow")
                .font(.system(size: 48))
                .foregroundStyle(.secondary)

            Text("{{PROJECT_NAME}}")
                .font(.title)

            Text("macOS アプリの土台が完成しました")
                .foregroundStyle(.secondary)
        }
        .frame(minWidth: 400, minHeight: 300)
        .padding()
    }
}

#Preview {
    ContentView()
}
