public class HelloShellix {
    public static void main(String[] args) {
        System.out.println("Hello from Shellix Java Runner! Args count: " + args.length);
        for (String a : args) {
            System.out.println("  arg: " + a);
        }
    }
}
