// WebView2 类型声明
interface WebView2MessageEvent {
  data: any;
}

interface WebView2 {
  postMessage(message: string): void;
  addEventListener(type: 'message', handler: (event: WebView2MessageEvent) => void): void;
  removeEventListener(type: 'message', handler: (event: WebView2MessageEvent) => void): void;
}

interface Chrome {
  webview: WebView2;
}

interface Window {
  chrome?: Chrome;
}
